---
id: compression-opportunity-test-specs
created: 2026-01-28T22:32:15Z
type: compression_opportunity
severity: medium
affected_specs:
  - test-file-organization
  - fix-hardcoded-paths-in-tests
  - optimize-test-performance
affected_files:
  - specs/test-file-organization/test-file-organization.md
  - specs/fix-hardcoded-paths-in-tests/fix-hardcoded-paths-in-tests.md
  - specs/optimize-test-performance/optimize-test-performance.md
overlap_type: domain
confidence: 0.75
original_specs:
  - id: test-file-organization
    file: specs/test-file-organization/test-file-organization.md
    title: Parser Test File Organization
  - id: fix-hardcoded-paths-in-tests
    file: specs/fix-hardcoded-paths-in-tests/fix-hardcoded-paths-in-tests.md
    title: Fix Hardcoded Paths in Tests
  - id: optimize-test-performance
    file: specs/optimize-test-performance/optimize-test-performance.md
    title: Optimize Test Performance
---

# Spec compression opportunity: test-file-organization + fix-hardcoded-paths-in-tests + optimize-test-performance

## Issue Description
Three separate specs all address different aspects of test infrastructure. These specs could be consolidated to reduce complexity and improve clarity. High-priority specs with overlapping scope may benefit from consolidation to avoid duplication of effort.

## Overlapping Specifications
- **test-file-organization**: Referenced in specs/test-file-organization/test-file-organization.md
- **fix-hardcoded-paths-in-tests**: Referenced in specs/fix-hardcoded-paths-in-tests/fix-hardcoded-paths-in-tests.md
- **optimize-test-performance**: Referenced in specs/optimize-test-performance/optimize-test-performance.md

## Overlap Analysis
**Type**: domain
**Confidence**: 75%

**Evidence**:
- All three specs focus on the test infrastructure domain
- They address complementary aspects: organization, portability, and performance
- All are priority 1 and not_started status
- Created within a week of each other (Jan 21-28, 2026)
- Could be consolidated into a single "test-infrastructure" spec

## Original Specification Details
### Parser Test File Organization
- **File**: specs/test-file-organization/test-file-organization.md
- **ID**: test-file-organization
- **Focus**: Organizing parser tests to avoid token limits and improve maintainability
- **Created**: 2026-01-21T23:30:00Z

### Fix Hardcoded Paths in Tests
- **File**: specs/fix-hardcoded-paths-in-tests/fix-hardcoded-paths-in-tests.md
- **ID**: fix-hardcoded-paths-in-tests
- **Focus**: Making tests portable by removing hardcoded absolute paths
- **Created**: 2026-01-22T20:45:00Z

### Optimize Test Performance
- **File**: specs/optimize-test-performance/optimize-test-performance.md
- **ID**: optimize-test-performance
- **Focus**: Getting test suite to run in under 10 seconds
- **Created**: 2026-01-28T21:25:00Z

## Impact
Important system behavior may not match specifications.

## Recommendation
**SPEC COMPRESSION OPPORTUNITY**: These specs have overlapping scope and could be consolidated:

1. Review the overlapping specs: test-file-organization, fix-hardcoded-paths-in-tests, optimize-test-performance
2. Analyze the overlap: All three specs focus on the test infrastructure domain; They address complementary aspects: organization, portability, and performance; All are priority 1 and not_started status; Created within a week of each other (Jan 21-28, 2026); Could be consolidated into a single "test-infrastructure" spec
3. Consider merging into a single comprehensive spec
4. Maintain traceability by referencing original specs in the consolidated version
5. Update any existing assertions to point to the new consolidated spec

**Confidence Level**: 75%
**Overlap Type**: domain