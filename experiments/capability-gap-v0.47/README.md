# WALDO v0.47 — capability gap detection hand

Status: **EXPERIMENTAL / TEST**

v0.47 vendors only the standalone AI-native Capability Gap Hand, not the browser Workbench UI. It compares explicit requirements against an explicit declared capability inventory and reports READY, DEGRADED, UNKNOWN or BLOCKED states plus bounded proposed contract requirements for genuinely missing capabilities.

A declaration is not runtime proof. Missing capability does not trigger installation, permission grants or silent quality reduction. The hand has no side effects and no authority; it makes absence visible so later routing or creation can respond deliberately instead of pretending the capability exists.
