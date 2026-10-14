package com.servicebay.billing.adapters.http;

import com.servicebay.billing.adapters.http.Dtos.ErrorDto;
import com.servicebay.billing.application.NotFoundException;
import com.servicebay.billing.domain.ports.ConflictException;
import com.servicebay.billing.domain.ports.Directories.UpstreamException;
import com.servicebay.billing.domain.ports.PaymentProviderException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.method.annotation.MethodArgumentTypeMismatchException;

/** Maps failures to the status codes and {"error": ...} body the Go services use. */
@RestControllerAdvice
class ErrorHandler {

    private static final Logger log = LoggerFactory.getLogger(ErrorHandler.class);

    @ExceptionHandler(NotFoundException.class)
    ResponseEntity<ErrorDto> notFound(NotFoundException e) {
        return error(HttpStatus.NOT_FOUND, e.getMessage());
    }

    @ExceptionHandler(ConflictException.class)
    ResponseEntity<ErrorDto> conflict(ConflictException e) {
        return error(HttpStatus.CONFLICT, e.getMessage());
    }

    @ExceptionHandler({IllegalArgumentException.class, HttpMessageNotReadableException.class,
            MethodArgumentTypeMismatchException.class})
    ResponseEntity<ErrorDto> badRequest(Exception e) {
        return error(HttpStatus.BAD_REQUEST, e instanceof IllegalArgumentException ? e.getMessage() : "malformed request");
    }

    @ExceptionHandler(PaymentProviderException.class)
    ResponseEntity<ErrorDto> provider(PaymentProviderException e) {
        HttpStatus status = switch (e.kind()) {
            case NOT_CONFIGURED -> HttpStatus.SERVICE_UNAVAILABLE;
            case CONFLICT -> HttpStatus.CONFLICT;
            case INVALID_WEBHOOK -> HttpStatus.BAD_REQUEST;
            case TEST_MODE_ONLY -> HttpStatus.FORBIDDEN;
            case UNAVAILABLE -> HttpStatus.BAD_GATEWAY;
        };
        return error(status, e.getMessage());
    }

    @ExceptionHandler(UpstreamException.class)
    ResponseEntity<ErrorDto> upstream(UpstreamException e) {
        log.warn("upstream call failed: {}", e.getMessage());
        return error(HttpStatus.BAD_GATEWAY, e.getMessage());
    }

    private static ResponseEntity<ErrorDto> error(HttpStatus status, String message) {
        return ResponseEntity.status(status).body(new ErrorDto(message));
    }
}
