namespace Subscriptions;

/// <summary>Subscription plan tiers.</summary>
public enum PlanTier { Free, Pro, Enterprise }

/// <summary>Represents a subscription plan with pricing and feature limits.</summary>
public sealed record Plan(
    string Id,
    string Name,
    PlanTier Tier,
    decimal MonthlyCostUsd,
    int MaxSeats)
{
    /// <summary>Returns true when the plan allows at least one additional seat.</summary>
    public bool HasAvailableSeat(int currentSeats) => currentSeats < MaxSeats;

    /// <summary>Calculates the annual cost including a 10% loyalty discount.</summary>
    public decimal AnnualCostUsd() => MonthlyCostUsd * 12 * 0.90m;

    /// <summary>Well-known plans.</summary>
    public static readonly Plan Free       = new("free",       "Free",       PlanTier.Free,       0m,    1);
    public static readonly Plan Pro        = new("pro",        "Pro",        PlanTier.Pro,        29m,   10);
    public static readonly Plan Enterprise = new("enterprise", "Enterprise", PlanTier.Enterprise, 299m, 500);
}
