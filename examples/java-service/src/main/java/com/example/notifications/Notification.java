package com.example.notifications;

import java.time.Instant;
import java.util.Objects;

/** Immutable notification message. */
public record Notification(
        String id,
        String recipientId,
        String subject,
        String body,
        Instant createdAt) {

    public Notification {
        Objects.requireNonNull(id, "id must not be null");
        Objects.requireNonNull(recipientId, "recipientId must not be null");
        if (subject == null || subject.isBlank()) throw new IllegalArgumentException("subject must not be blank");
        Objects.requireNonNull(body, "body must not be null");
        Objects.requireNonNull(createdAt, "createdAt must not be null");
    }

    /** Returns a copy with a trimmed subject. */
    public Notification withTrimmedSubject() {
        return new Notification(id, recipientId, subject.trim(), body, createdAt);
    }
}
