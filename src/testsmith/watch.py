"""
Watch mode for TestSmith - automatically regenerate tests on file changes.
"""
import time
from pathlib import Path
from datetime import datetime
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler, FileSystemEvent

from testsmith.support.config import TestSmithConfig
from testsmith.core.project_detector import build_project_context


class DebounceHandler(FileSystemEventHandler):
    """File system event handler with debouncing."""
    
    def __init__(self, callback, debounce_seconds: float = 0.5):
        """
        Initialize handler.
        
        Args:
            callback: Function to call when a file changes (receives file path)
            debounce_seconds: Minimum time between processing events for the same file
        """
        super().__init__()
        self.callback = callback
        self.debounce_seconds = debounce_seconds
        self.last_processed = {}
    
    def on_modified(self, event: FileSystemEvent):
        """Handle file modification events."""
        if event.is_directory:
            return
        
        if not event.src_path.endswith('.py'):
            return
        
        self._process_event(event.src_path)
    
    def on_created(self, event: FileSystemEvent):
        """Handle file creation events."""
        if event.is_directory:
            return
        
        if not event.src_path.endswith('.py'):
            return
        
        self._process_event(event.src_path)
    
    def _process_event(self, file_path: str):
        """Process a file event with debouncing."""
        now = time.time()
        last_time = self.last_processed.get(file_path, 0)
        
        # Debounce: skip if processed recently
        if now - last_time < self.debounce_seconds:
            return
        
        self.last_processed[file_path] = now
        self.callback(Path(file_path))


def watch_project(project_root: Path, config: TestSmithConfig, process_file_func):
    """
    Watch project for file changes and regenerate tests.
    
    Args:
        project_root: Project root directory
        config: TestSmith configuration
        process_file_func: Function to call to process a changed file
    """
    def on_file_change(file_path: Path):
        """Handle file change event."""
        timestamp = datetime.now().strftime("%H:%M:%S")
        
        # Skip test files
        if file_path.name.startswith("test_"):
            return
        
        # Skip excluded directories
        if any(excluded in file_path.parts for excluded in config.exclude_dirs):
            return
        
        # Skip test directory
        test_root_name = config.test_root.rstrip("/").split("/")[-1]
        if test_root_name in file_path.parts:
            return
        
        try:
            rel_path = file_path.relative_to(project_root)
        except ValueError:
            # File is outside project root
            return
        
        print(f"[{timestamp}] Detected change: {rel_path} → regenerating...")
        
        try:
            process_file_func(file_path)
            print(f"[{timestamp}] ✓ Test updated for {rel_path}")
        except Exception as e:
            print(f"[{timestamp}] ✗ Error processing {rel_path}: {e}")
    
    # Create event handler
    handler = DebounceHandler(on_file_change, debounce_seconds=0.5)
    
    # Create observer
    observer = Observer()
    
    # Watch project root recursively
    observer.schedule(handler, str(project_root), recursive=True)
    
    print(f"👀 Watching {project_root} for changes...")
    print("Press Ctrl+C to stop watching.")
    print()
    
    observer.start()
    
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n\nStopping watch mode...")
        observer.stop()
    
    observer.join()
    print("Watch mode stopped.")
