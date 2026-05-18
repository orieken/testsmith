package com.example.notifications;

/** Port for delivering notifications to an external channel. */
public interface NotificationSender {
    void deliver(Notification notification);
}
