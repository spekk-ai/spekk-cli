---
id: business-model-validator
created: 2026-01-23T22:14:00Z
priority: 2
---

# Business Model Validator

Validates startup/business models through structured questioning and scoring across key areas like product validation, market demand, and business strategy.

## Triggers

- "business model"
- "startup"
- "validate"
- "business idea"
- "startup idea"
- "business validation"
- "market validation"
- "product validation"
- "business strategy"
- "business plan"
- "monetization"
- "revenue model"
- "traction"
- "market demand"

## Workflow

1. Ask about industry background and founder expertise
2. Assess product validation status and user testing
3. Evaluate market demand evidence and target customer clarity
4. Review business goals and monetization strategy
5. Examine traction metrics and current momentum
6. Identify risky assumptions and hypothesis prioritization
7. Score each area (0-2: concerning/adequate/clear)
8. Generate recommendations based on low-scoring areas
9. Create structured markdown report with overall health score
10. Save report to projects/{project-name}/business-model-validation.md
11. Present summary with prioritized next steps

## Validation

- All six assessment areas covered (industry-background, product-validation, market-demand, business-goals, traction-momentum, hypothesis-prioritization)
- Each area scored from 0-2
- Overall health score calculated (out of 12 maximum)
- High-priority recommendations generated for areas scoring 0
- Medium-priority recommendations generated for areas scoring 1
- Report saved successfully to projects directory
- Report includes strengths, concerns, priority hypotheses, and recommended actions

## Assessment Areas

### Industry Background
- Founder's relevant experience
- Years in industry
- Domain expertise
- Prior product experience

### Product Validation
- Development stage
- User testing conducted
- Feedback received
- Prototype/MVP status

### Market Demand
- Evidence of demand
- Target customer definition
- Problem being solved
- Market size

### Business Goals
- Primary business goal
- Monetization strategy
- Revenue generation plan
- Key success metrics

### Traction & Momentum
- Current metrics
- Funding raised
- Paying customers
- Recent growth

### Hypothesis Prioritization
- Riskiest assumptions
- Business model invalidation risks
- Priority testing sequence
- Validation approach

## Output Format

Report structure:
```
BUSINESS MODEL VALIDATION REPORT
================================
Overall Health Score: X/12 points (X%)

Strengths:
- [Areas scoring 2]

Concerns:
- [Areas scoring 0]

Unanswered/Unclear:
- [Areas scoring 1]

Priority Hypotheses to Test (Next 6 Months):
1. [Top hypothesis from high-priority recommendations]
2. [Second hypothesis]
3. [Third hypothesis]

Recommended Actions:
- [Action for each non-perfect area]
```
