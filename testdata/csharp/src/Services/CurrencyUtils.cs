using System;
using System.Globalization;

namespace MyApp.Services;

public static class CurrencyUtils
{
    public static double CalculateTotal(double[] items, double taxRate)
    {
        double total = 0.0;
        foreach (var item in items)
            total += item;
        return total * (1 + taxRate);
    }

    public static string FormatAmount(double amount, string currencyCode)
    {
        var culture = new CultureInfo("en-US");
        return amount.ToString("C", culture) + " " + currencyCode;
    }
}
