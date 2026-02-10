"""
Unit tests for watch module.
"""

import time
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock
from watchdog.events import FileModifiedEvent, FileCreatedEvent, DirModifiedEvent

from testsmith.watch import DebounceHandler, watch_project
from testsmith.support.config import TestSmithConfig as Config


def test_debounce_handler_ignores_directories():
    """Test that handler ignores directory events."""
    callback = Mock()
    handler = DebounceHandler(callback, debounce_seconds=0.1)

    event = DirModifiedEvent("/some/directory")
    handler.on_modified(event)

    callback.assert_not_called()


def test_debounce_handler_ignores_non_python_files():
    """Test that handler ignores non-Python files."""
    callback = Mock()
    handler = DebounceHandler(callback, debounce_seconds=0.1)

    event = FileModifiedEvent("/some/file.txt")
    handler.on_modified(event)

    callback.assert_not_called()


def test_debounce_handler_processes_python_files():
    """Test that handler processes Python file modifications."""
    callback = Mock()
    handler = DebounceHandler(callback, debounce_seconds=0.1)

    event = FileModifiedEvent("/some/file.py")
    handler.on_modified(event)

    callback.assert_called_once_with(Path("/some/file.py"))


def test_debounce_handler_processes_created_files():
    """Test that handler processes Python file creations."""
    callback = Mock()
    handler = DebounceHandler(callback, debounce_seconds=0.1)

    event = FileCreatedEvent("/some/new_file.py")
    handler.on_created(event)

    callback.assert_called_once_with(Path("/some/new_file.py"))


def test_debounce_handler_debounces_rapid_changes():
    """Test that handler debounces rapid successive changes."""
    callback = Mock()
    handler = DebounceHandler(callback, debounce_seconds=0.2)

    event = FileModifiedEvent("/some/file.py")

    # First event should be processed
    handler.on_modified(event)
    assert callback.call_count == 1

    # Immediate second event should be debounced
    handler.on_modified(event)
    assert callback.call_count == 1

    # After debounce period, should be processed again
    time.sleep(0.25)
    handler.on_modified(event)
    assert callback.call_count == 2


def test_debounce_handler_different_files_not_debounced():
    """Test that different files are not debounced together."""
    callback = Mock()
    handler = DebounceHandler(callback, debounce_seconds=0.2)

    event1 = FileModifiedEvent("/some/file1.py")
    event2 = FileModifiedEvent("/some/file2.py")

    handler.on_modified(event1)
    handler.on_modified(event2)

    assert callback.call_count == 2


@patch("testsmith.watch.Observer")
def test_watch_project_starts_observer(mock_observer_class):
    """Test that watch_project starts the observer."""
    mock_observer = MagicMock()
    mock_observer_class.return_value = mock_observer

    config = Config()
    project_root = Path("/fake/project")
    process_func = Mock()

    # Mock the observer to stop immediately
    def stop_after_start(*args, **kwargs):
        raise KeyboardInterrupt()

    mock_observer.start.side_effect = stop_after_start

    watch_project(project_root, config, process_func)

    mock_observer.schedule.assert_called_once()
    mock_observer.start.assert_called_once()
    mock_observer.stop.assert_called_once()
    mock_observer.join.assert_called_once()


@patch("testsmith.watch.Observer")
def test_watch_project_handles_keyboard_interrupt(mock_observer_class):
    """Test that watch_project handles Ctrl+C gracefully."""
    mock_observer = MagicMock()
    mock_observer_class.return_value = mock_observer

    config = Config()
    project_root = Path("/fake/project")
    process_func = Mock()

    # Simulate Ctrl+C after start
    def raise_interrupt(*args, **kwargs):
        raise KeyboardInterrupt()

    mock_observer.start.side_effect = raise_interrupt

    # Should not raise exception
    watch_project(project_root, config, process_func)

    mock_observer.stop.assert_called_once()


def test_debounce_handler_custom_debounce_time():
    """Test that custom debounce time is respected."""
    callback = Mock()
    # Use a longer debounce time to make the test more reliable
    handler = DebounceHandler(callback, debounce_seconds=0.2)

    event = FileModifiedEvent("/some/file.py")

    # First event - should trigger callback
    handler.on_modified(event)
    assert callback.call_count == 1

    # Second event within debounce window - should be ignored
    time.sleep(0.1)  # Half the debounce time
    handler.on_modified(event)
    assert callback.call_count == 1, "Event within debounce window should be ignored"

    # Third event after debounce window - should trigger callback
    time.sleep(0.25)  # More than debounce time from first event
    handler.on_modified(event)
    assert (
        callback.call_count == 2
    ), "Event after debounce window should trigger callback"
