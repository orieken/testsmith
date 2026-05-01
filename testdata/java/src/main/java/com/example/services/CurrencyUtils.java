package com.example.services;

import java.text.NumberFormat;
import java.util.Currency;
import java.util.List;

public class CurrencyUtils {

    public static double calculateTotal(List<Double> items, double taxRate) {
        double total = 0.0;
        for (double item : items) {
            total += item;
        }
        return total * (1 + taxRate);
    }

    public static String formatCurrency(double amount, String currencyCode) {
        Currency currency = Currency.getInstance(currencyCode);
        NumberFormat format = NumberFormat.getCurrencyInstance();
        format.setCurrency(currency);
        return format.format(amount);
    }
}
