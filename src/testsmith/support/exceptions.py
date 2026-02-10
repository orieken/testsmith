"""
Custom exceptions for TestSmith.
"""

class TestSmithError(Exception):
    """Base exception for all TestSmith errors."""
    pass


class ProjectRootNotFoundError(TestSmithError):
    """Raised when the project root cannot be detected."""
    pass


class SourceParseError(TestSmithError):
    """Raised when source code cannot be parsed."""
    def __init__(self, file_path: str, line_number: int, message: str):
        self.file_path = file_path
        self.line_number = line_number
        self.message = message
        super().__init__(f"Syntax error in {file_path} at line {line_number}: {message}")
