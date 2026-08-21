package com.example.notifications;

import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;

/** Manages notification creation and delivery. */
public class NotificationService {

    private final NotificationSender sender;
    private final Map<String, Notification> store = new ConcurrentHashMap<>();

    public NotificationService(NotificationSender sender) {
        this.sender = sender;
    }

    /**
     * Creates and sends a notification.
     *
     * @throws IllegalArgumentException when recipientId or subject is blank
     */
    public Notification send(String recipientId, String subject, String body) {
        if (recipientId == null || recipientId.isBlank()) {
            throw new IllegalArgumentException("recipientId must not be blank");
        }
        Notification n = new Notification(
                UUID.randomUUID().toString(),
                recipientId,
                subject,
                body,
                Instant.now());
        store.put(n.id(), n);
        sender.deliver(n);
        return n;
    }

    /** Returns all notifications sent to a given recipient. */
    public List<Notification> findByRecipient(String recipientId) {
        List<Notification> result = new ArrayList<>();
        for (Notification n : store.values()) {
            if (n.recipientId().equals(recipientId)) result.add(n);
        }
        return result;
    }

    /** Returns total notifications in the store. */
    public int count() {
        return store.size();
    }
}
