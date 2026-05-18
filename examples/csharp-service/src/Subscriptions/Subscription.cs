namespace Subscriptions;

/// <summary>Represents a tenant's active or cancelled subscription.</summary>
public sealed record Subscription(
    string Id,
    string TenantId,
    Plan Plan,
    DateTimeOffset CreatedAt,
    DateTimeOffset? UpdatedAt = null,
    DateTimeOffset? CancelledAt = null)
{
    /// <summary>Returns true when the subscription has not been cancelled.</summary>
    public bool IsActive => CancelledAt is null;
}
