package com.example.services;

import java.util.List;
import java.util.Optional;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;
import com.example.domain.Payment;

@Service
public class PaymentService {

    private final String apiKey;
    private final RestTemplate restTemplate;

    public PaymentService(String apiKey, RestTemplate restTemplate) {
        this.apiKey = apiKey;
        this.restTemplate = restTemplate;
    }

    public Payment charge(int amount, String token) {
        if (amount <= 0) {
            throw new IllegalArgumentException("Amount must be positive");
        }
        return new Payment(amount, token);
    }

    public boolean refund(String chargeId) {
        if (chargeId == null || chargeId.isEmpty()) {
            return false;
        }
        return true;
    }

    private void internalHelper() {}
}
