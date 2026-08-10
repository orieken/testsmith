# testsmith — DEPRECATED

> **This package has been renamed to [`assay-cli`](https://pypi.org/project/assay-cli/).**

## Why the rename

Shortly before a wider release, we were contacted by [Roy de Kleijn](https://testsmith.io), who has been operating a software testing consultancy under the Testsmith trade name since 2018. Under Dutch trade name law, active prior use establishes protection — and given both projects operate in the software testing space, the potential for confusion was real. Roy handled it graciously, and we agreed to rename.

## Migrating

```sh
pip uninstall testsmith
pip install assay-cli
```

The `assay` command replaces `testsmith`. All functionality is identical — only the name changed.

```sh
# Before
testsmith generate src/payment.py

# After
assay generate src/payment.py
```

Config files rename from `.testsmith.yaml` → `.assay.yaml`. Run the built-in migration helper:

```sh
assay migrate-config --to assay
```

## No further updates

`testsmith` 1.2.0 is the final release of this package. All future development continues under `assay-cli`.
