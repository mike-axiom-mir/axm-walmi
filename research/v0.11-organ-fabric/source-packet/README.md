# v0.11 source packet

The original donor ZIP is published here as four content-addressed binary parts because the connector publication path cannot attach the ZIP as a single binary file.

Reconstruct it with:

```bash
python reassemble_source_packet.py
```

Expected result:

`sha256:8f4688db3abcba988011f2a2f36f172a09c3d848e57a2da54db90abd2669e634`

The packet contains separate/unmerged Code Fabric, Deterministic Organ Fabric, and Organ Archive source sets captured before their local review branches were publicly reachable.
