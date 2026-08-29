# WALDO v0.48 — evidence routing hand

Status: **EXPERIMENTAL / TEST**

v0.48 vendors the standalone AI-native Evidence Router Hand. It does not decide whether a claim is true. It maps a declared claim kind and risk to the primary evidence surface, counterevidence shape, and—when risk is high—a secondary surface that should test the claim.

The route matrix covers existence, structure, behavior, visual, temporal, interaction, persistence, transport, authorization, performance, resource safety, learning, quality and steward/taste claims. Unknown kinds are held as `UNKNOWN_KIND` and require a domain-native verifier instead of receiving an invented verdict.

Every route starts `UNTESTED`; automatic verdict and automatic action are false. Evidence routing is guidance for verification, not proof or authority.
