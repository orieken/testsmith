namespace Subscriptions;

/// <summary>Manages tenant subscriptions.</summary>
public sealed class SubscriptionService
{
    private readonly Dictionary<string, Subscription> _store = new();

    /// <summary>Creates a new subscription for a tenant on the Free plan.</summary>
    /// <exception cref="InvalidOperationException">Thrown when tenantId already has a subscription.</exception>
    public Subscription Subscribe(string tenantId, Plan plan)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(tenantId);
        ArgumentNullException.ThrowIfNull(plan);

        if (_store.ContainsKey(tenantId))
            throw new InvalidOperationException($"Tenant '{tenantId}' already has a subscription.");

        var sub = new Subscription(Guid.NewGuid().ToString(), tenantId, plan, DateTimeOffset.UtcNow);
        _store[tenantId] = sub;
        return sub;
    }

    /// <summary>Upgrades an existing subscription to a new plan.</summary>
    /// <exception cref="KeyNotFoundException">Thrown when the tenant has no subscription.</exception>
    public Subscription Upgrade(string tenantId, Plan newPlan)
    {
        if (!_store.TryGetValue(tenantId, out var existing))
            throw new KeyNotFoundException($"No subscription found for tenant '{tenantId}'.");

        var upgraded = existing with { Plan = newPlan, UpdatedAt = DateTimeOffset.UtcNow };
        _store[tenantId] = upgraded;
        return upgraded;
    }

    /// <summary>Cancels a tenant's subscription.</summary>
    public bool Cancel(string tenantId)
    {
        if (!_store.TryGetValue(tenantId, out var sub)) return false;
        _store[tenantId] = sub with { CancelledAt = DateTimeOffset.UtcNow };
        return true;
    }

    /// <summary>Returns the current subscription for a tenant, or null when none exists.</summary>
    public Subscription? Find(string tenantId) =>
        _store.TryGetValue(tenantId, out var sub) ? sub : null;

    /// <summary>Returns all active (non-cancelled) subscriptions.</summary>
    public IReadOnlyList<Subscription> ListActive() =>
        _store.Values.Where(s => s.CancelledAt is null).ToList();
}
