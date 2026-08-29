# ADR 9041: Completion is not the learning gate

## Status

Experimental.

## Context

The existing positive-target path correctly refuses to teach a model to imitate harmful or inconclusive reactions. It incorrectly became the only weight-learning path, which made completion or correction a practical prerequisite for learning.

## Decision

Retain every observed attempt and outcome. Helpful or corrected outputs may remain positive response targets. Incomplete, harmful, and inconclusive attempts may instead produce an outcome-conditioned reflection trajectory:

- the original request is context;
- the attempted action and observed outcome are tool evidence;
- the failed reaction is never a supervised assistant target;
- the visible lesson is the only supervised assistant target;
- completion is not required.

This projection uses the existing `assistant-response-modeling` objective and remains bound to the observed source receipt.

## Consequence

WALDO can learn how to diagnose, resume, and change strategy after failure without being trained to repeat the failure. A projection still does not claim that training ran or that weights improved.

## Boundary

`EXPERIENCE_CAN_TEACH_WITHOUT_COMPLETION`

Learning eligibility does not grant execution, training, promotion, merge, CANON, or world-action authority.
