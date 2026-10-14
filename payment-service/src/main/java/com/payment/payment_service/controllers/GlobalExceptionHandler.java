package com.payment.payment_service.controllers;

import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import com.payment.payment_service.controllers.dto.ErrorResponse;
import com.payment.payment_service.exceptions.ConflictException;
import com.payment.payment_service.exceptions.GatewayException;
import com.payment.payment_service.exceptions.GatewayNotConfiguredException;
import com.payment.payment_service.exceptions.InvalidWebhookException;
import com.payment.payment_service.exceptions.NotFoundException;
import com.payment.payment_service.exceptions.TestModeOnlyException;

@RestControllerAdvice
public class GlobalExceptionHandler {

  @ExceptionHandler(NotFoundException.class)
  @ResponseStatus(HttpStatus.NOT_FOUND)
  public ErrorResponse handleNotFound(NotFoundException e) {
    return new ErrorResponse(e.getMessage());
  }

  @ExceptionHandler(ConflictException.class)
  @ResponseStatus(HttpStatus.CONFLICT)
  public ErrorResponse handleConflict(ConflictException e) {
    return new ErrorResponse(e.getMessage());
  }

  // Lost race on the billing_payment_id UNIQUE constraint.
  @ExceptionHandler(DataIntegrityViolationException.class)
  @ResponseStatus(HttpStatus.CONFLICT)
  public ErrorResponse handleIntegrity(DataIntegrityViolationException e) {
    return new ErrorResponse("request conflicts with existing data");
  }

  @ExceptionHandler(GatewayException.class)
  public ResponseEntity<ErrorResponse> handleGateway(GatewayException e) {
    return ResponseEntity.status(e.isConflict() ? HttpStatus.CONFLICT : HttpStatus.BAD_GATEWAY)
        .body(new ErrorResponse(e.getMessage()));
  }

  @ExceptionHandler(GatewayNotConfiguredException.class)
  @ResponseStatus(HttpStatus.SERVICE_UNAVAILABLE)
  public ErrorResponse handleNotConfigured(GatewayNotConfiguredException e) {
    return new ErrorResponse(e.getMessage());
  }

  @ExceptionHandler(TestModeOnlyException.class)
  @ResponseStatus(HttpStatus.FORBIDDEN)
  public ErrorResponse handleTestModeOnly(TestModeOnlyException e) {
    return new ErrorResponse(e.getMessage());
  }

  @ExceptionHandler(InvalidWebhookException.class)
  @ResponseStatus(HttpStatus.BAD_REQUEST)
  public ErrorResponse handleInvalidWebhook(InvalidWebhookException e) {
    return new ErrorResponse(e.getMessage());
  }

  @ExceptionHandler({ IllegalArgumentException.class, HttpMessageNotReadableException.class })
  @ResponseStatus(HttpStatus.BAD_REQUEST)
  public ErrorResponse handleBadRequest(Exception e) {
    return new ErrorResponse(e instanceof IllegalArgumentException ? e.getMessage() : "malformed request body");
  }
}
