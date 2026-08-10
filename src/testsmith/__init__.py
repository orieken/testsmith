"""testsmith — renamed to assay-cli.

This package has been renamed. Please migrate:

    pip uninstall testsmith
    pip install assay-cli

See https://github.com/orieken/assay for details.
"""

import warnings

warnings.warn(
    "The 'testsmith' package has been renamed to 'assay-cli'. "
    "Please run: pip uninstall testsmith && pip install assay-cli. "
    "This package will receive no further updates.",
    DeprecationWarning,
    stacklevel=2,
)

__version__ = "1.2.0"
