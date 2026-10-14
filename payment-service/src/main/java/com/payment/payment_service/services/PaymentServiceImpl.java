package com.payment.payment_service.services;

import java.util.Optional;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

import com.payment.payment_service.entity.PaymentEntity;
import com.payment.payment_service.entity.PaymentStatus;
import com.payment.payment_service.events.PaymentEvents;
import com.payment.payment_service.exceptions.ConflictException;
import com.payment.payment_service.exceptions.NotFoundException;
import com.payment.payment_service.exceptions.TestModeOnlyException;
import com.payment.payment_service.gateway.Money;
import com.payment.payment_service.ports.PaymentGateway;
import com.payment.payment_service.ports.PaymentGateway.GatewayIntent;
import com.payment.payment_service.ports.PaymentGateway.WebhookEvent;
import com.payment.payment_service.ports.PaymentService;
import com.payment.payment_service.repositories.OutboxRepository;
import com.payment.payment_service.repositories.PaymentRepository;
import com.payment.payment_service.repositories.StripeEventRepository;

// Transactions are opened explicitly (TransactionOperations) rather than with
// @Transactional, because no DB transaction or row lock may be held across a
// call to Stripe: every gateway call sits between two short transactions.
@Service
public class PaymentServiceImpl implements PaymentService {

  private static final Logger log = LoggerFactory.getLogger(PaymentServiceImpl.class);

  // Stripe's test card that always succeeds; pm_card_chargeDeclined always declines.
  private static final String DEFAULT_TEST_PAYMENT_METHOD = "pm_card_visa";
  // Stripe requires a return_url when confirming server-side; test cards never redirect.
  private static final String TEST_RETURN_URL = "https://example.com/stripe-test-return";

  private final PaymentRepository paymentRepository;
  private final StripeEventRepository stripeEventRepository;
  private final OutboxRepository outboxRepository;
  private final PaymentGateway gateway;
  private final TransactionOperations tx;

  public PaymentServiceImpl(PaymentRepository paymentRepository, StripeEventRepository stripeEventRepository,
      OutboxRepository outboxRepository, PaymentGateway gateway, TransactionOperations tx) {
    this.paymentRepository = paymentRepository;
    this.stripeEventRepository = stripeEventRepository;
    this.outboxRepository = outboxRepository;
    this.gateway = gateway;
    this.tx = tx;
  }

  @Override
  public Checkout createPayment(CreatePaymentCommand cmd) {
    String currency = Money.normalizeCurrency(cmd.currency());
    Money.toMinorUnits(cmd.amount(), currency); // reject unrepresentable amounts before writing anything

    PaymentEntity payment = tx.execute(status -> paymentRepository.findByBillingPaymentId(cmd.billingPaymentId())
        .map(existing -> requireSameRequest(existing, cmd, currency))
        .orElseGet(() -> paymentRepository.save(new PaymentEntity(
            cmd.billId(), cmd.billingPaymentId(), cmd.customerId(), cmd.amount(), currency))));

    switch (payment.getStatus()) {
      case SUCCEEDED -> {
        return new Checkout(payment, null);
      }
      case CANCELED -> throw new ConflictException("payment " + payment.getId() + " was canceled");
      default -> {
      }
    }

    if (payment.getStripePaymentIntentId() != null) {
      // Retry after a successful create: hand back the same intent's secret.
      return new Checkout(payment, gateway.retrieveIntent(payment.getStripePaymentIntentId()).clientSecret());
    }

    GatewayIntent intent = gateway.createIntent(payment.getId(), payment.getBillId(), payment.getAmount(), currency);

    PaymentEntity linked = tx.execute(status -> {
      PaymentEntity locked = lock(payment.getId());
      if (locked.getStripePaymentIntentId() == null) {
        locked.setStripePaymentIntentId(intent.id());
        paymentRepository.save(locked);
      }
      return locked;
    });

    if (linked.getStatus() == PaymentStatus.CANCELED) {
      // Canceled while Stripe was creating the intent: don't leave a payable intent behind.
      gateway.cancelIntent(intent.id());
      throw new ConflictException("payment " + linked.getId() + " was canceled");
    }
    return new Checkout(linked, intent.clientSecret());
  }

  @Override
  public PaymentEntity getPayment(Long id) {
    return paymentRepository.findById(id)
        .orElseThrow(() -> new NotFoundException("payment " + id + " not found"));
  }

  @Override
  public PaymentEntity cancelPayment(Long id) {
    PaymentEntity payment = getPayment(id);
    if (payment.getStatus() == PaymentStatus.CANCELED) {
      return payment;
    }
    if (payment.getStatus() == PaymentStatus.SUCCEEDED) {
      throw new ConflictException("payment " + id + " already succeeded");
    }

    // Stripe refuses to cancel an intent that already succeeded or is
    // processing; that surfaces as a 409 GatewayException and nothing changes here.
    if (payment.getStripePaymentIntentId() != null) {
      gateway.cancelIntent(payment.getStripePaymentIntentId());
    }

    return tx.execute(status -> {
      PaymentEntity locked = lock(id);
      if (locked.getStatus().isTerminal()) {
        return locked; // the canceled webhook got here first
      }
      locked.setStatus(PaymentStatus.CANCELED);
      locked.setFailureReason("canceled by billing-service");
      paymentRepository.save(locked);
      outboxRepository.save(PaymentEvents.failed(locked));
      return locked;
    });
  }

  @Override
  public void handleWebhook(WebhookEvent event) {
    PaymentStatus target = switch (event.type()) {
      case "payment_intent.succeeded" -> PaymentStatus.SUCCEEDED;
      case "payment_intent.processing" -> PaymentStatus.PROCESSING;
      case "payment_intent.payment_failed" -> PaymentStatus.PENDING; // attempt failed, customer may retry
      case "payment_intent.canceled" -> PaymentStatus.CANCELED;
      default -> null;
    };
    if (target == null) {
      log.debug("ignoring Stripe event {} of type {}", event.id(), event.type());
      return;
    }

    tx.executeWithoutResult(status -> {
      if (stripeEventRepository.markProcessed(event.id(), event.type()) == 0) {
        log.info("Stripe event {} already processed", event.id());
        return;
      }

      Optional<PaymentEntity> found = event.paymentId() != null
          ? paymentRepository.findByIdForUpdate(event.paymentId())
          : Optional.ofNullable(event.intentId()).flatMap(paymentRepository::findByStripePaymentIntentIdForUpdate);
      if (found.isEmpty()) {
        // Not an intent this service created (another integration on the same Stripe account).
        log.warn("Stripe event {} ({}) references unknown payment intent {}", event.id(), event.type(),
            event.intentId());
        return;
      }

      applyStatus(found.get(), target, event.intentId(), event.amountReceived(), event.failureReason(),
          "Stripe event " + event.id());
    });
  }

  @Override
  public PaymentEntity syncPayment(Long id) {
    PaymentEntity payment = getPayment(id);
    if (payment.getStripePaymentIntentId() == null) {
      throw new ConflictException("payment " + id + " has no Stripe payment intent yet");
    }
    return applyIntent(id, gateway.retrieveIntent(payment.getStripePaymentIntentId()));
  }

  @Override
  public PaymentEntity confirmTestPayment(Long id, String paymentMethod) {
    if (!gateway.isTestMode()) {
      throw new TestModeOnlyException("test-confirm needs a Stripe test-mode key (sk_test_...)");
    }
    PaymentEntity payment = getPayment(id);
    if (payment.getStatus().isTerminal()) {
      throw new ConflictException("payment " + id + " is already " + payment.getStatus());
    }
    if (payment.getStripePaymentIntentId() == null) {
      throw new ConflictException("payment " + id + " has no Stripe payment intent yet");
    }
    GatewayIntent intent = gateway.confirmIntent(payment.getStripePaymentIntentId(),
        paymentMethod == null || paymentMethod.isBlank() ? DEFAULT_TEST_PAYMENT_METHOD : paymentMethod,
        TEST_RETURN_URL);
    return applyIntent(id, intent);
  }

  private PaymentEntity applyIntent(Long id, GatewayIntent intent) {
    return tx.execute(status -> {
      PaymentEntity locked = lock(id);
      applyStatus(locked, fromStripeStatus(intent.status()), intent.id(), intent.amountReceived(),
          intent.failureReason(), "sync of " + intent.id());
      return locked;
    });
  }

  // Shared by webhooks and sync, so both paths apply the same rules. Runs inside a
  // transaction with the payment row locked.
  private void applyStatus(PaymentEntity payment, PaymentStatus target, String intentId, Long amountReceived,
      String failureReason, String source) {
    if (payment.getStatus().isTerminal()) {
      // Stripe doesn't guarantee event order; a terminal state never moves.
      log.info("payment {} is {}, ignoring {} ({})", payment.getId(), payment.getStatus(), source, target);
      return;
    }
    if (payment.getStripePaymentIntentId() == null) {
      payment.setStripePaymentIntentId(intentId);
    }

    payment.setStatus(target);
    switch (target) {
      case SUCCEEDED -> {
        payment.setFailureReason(null);
        checkAmount(payment, amountReceived);
        outboxRepository.save(PaymentEvents.succeeded(payment));
      }
      case CANCELED -> {
        payment.setFailureReason(failureReason != null ? failureReason : "canceled");
        outboxRepository.save(PaymentEvents.failed(payment));
      }
      case PENDING -> payment.setFailureReason(failureReason);
      case PROCESSING -> {
      }
    }
    paymentRepository.save(payment);
  }

  // requires_payment_method / requires_confirmation / requires_action all mean the
  // customer still has to act.
  static PaymentStatus fromStripeStatus(String stripeStatus) {
    return switch (stripeStatus) {
      case "succeeded" -> PaymentStatus.SUCCEEDED;
      case "processing" -> PaymentStatus.PROCESSING;
      case "canceled" -> PaymentStatus.CANCELED;
      default -> PaymentStatus.PENDING;
    };
  }

  private PaymentEntity lock(Long id) {
    return paymentRepository.findByIdForUpdate(id)
        .orElseThrow(() -> new NotFoundException("payment " + id + " not found"));
  }

  // billing-service may retry a create; a retry has to describe the same charge.
  private static PaymentEntity requireSameRequest(PaymentEntity existing, CreatePaymentCommand cmd, String currency) {
    if (!existing.getBillId().equals(cmd.billId())
        || existing.getAmount().compareTo(cmd.amount()) != 0
        || !existing.getCurrency().equals(currency)) {
      throw new ConflictException("billing payment " + cmd.billingPaymentId()
          + " already exists with a different bill, amount or currency");
    }
    return existing;
  }

  // The money was taken either way, so a mismatch is logged loudly, not rejected.
  private static void checkAmount(PaymentEntity payment, Long amountReceived) {
    long expected = Money.toMinorUnits(payment.getAmount(), payment.getCurrency());
    if (amountReceived != null && amountReceived != expected) {
      log.error("payment {} succeeded with amount_received {} but {} {} minor units were expected",
          payment.getId(), amountReceived, expected, payment.getCurrency());
    }
  }
}
