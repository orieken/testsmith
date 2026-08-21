using System;
using System.Threading.Tasks;
using Stripe;
using Microsoft.Extensions.Logging;

namespace MyApp.Services;

public class PaymentService
{
    private readonly ILogger<PaymentService> _logger;
    private readonly string _apiKey;

    public PaymentService(ILogger<PaymentService> logger, string apiKey)
    {
        _logger = logger;
        _apiKey = apiKey;
    }

    public async Task<string> ChargeAsync(int amount, string token)
    {
        if (amount <= 0)
            throw new ArgumentException("Amount must be positive", nameof(amount));

        _logger.LogInformation("Charging {Amount}", amount);
        return Guid.NewGuid().ToString();
    }

    public bool Refund(string chargeId)
    {
        if (string.IsNullOrEmpty(chargeId))
            return false;

        return true;
    }

    private void InternalHelper() { }
}
