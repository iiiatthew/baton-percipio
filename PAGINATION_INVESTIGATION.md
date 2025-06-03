# Content-Discovery Pagination Investigation

## Problem Summary

The baton-percipio connector was making **5700 API calls per hour** to the content-discovery endpoint, far exceeding expected usage for a connector that runs once per hour.

**Expected**: 1-20 calls per sync  
**Actual**: 5700 calls per hour  
**Impact**: API rate limiting, excessive resource usage

## Root Cause Analysis

Investigation revealed four critical issues:

### 1. Mixed Pagination Patterns

Both APIs were incorrectly using the same pagination functions, but they have different documented patterns:

**User Management API** (simple pattern):

- Uses `offset` + `max` parameters
- Documents `x-total-count` header in swagger
- No `pagingRequestId` required
- Stateless pagination

**Content Discovery API** (complex pattern):

- Uses `offset` + `max` + `pagingRequestId` parameters
- Does NOT document `x-total-count` header in swagger
- Requires `pagingRequestId` for consistency
- Stateful pagination

### 2. No Infinite Loop Protection

No maximum page limit existed to prevent runaway pagination if API returned corrupted total counts.

### 3. Inadequate Header Validation

Missing comprehensive validation for pagination headers, especially for content-discovery API where `x-total-count` is not documented.

### 4. Missing API Behavior Validation

No consistency checks during pagination (e.g., lost `pagingRequestId`, empty responses when more data expected).

## Implemented Solutions

### Safety Limits

Added `MaxPagesPerSync = 100` to prevent infinite pagination regardless of API issues.

### Separate Pagination Patterns

- **Users**: `ParseSimplePaginationToken()` and `GetSimpleNextToken()` - generates `{"offset":1000}`
- **Courses**: `ParsePaginationToken()` and `GetNextToken()` - generates `{"pagingRequestId":"uuid","offset":1000}`

### Robust Header Handling

- Enhanced `getTotalCount()` with comprehensive validation
- Added `getTotalCountForContentDiscovery()` with estimation fallback for missing headers
- Sanity checks reject unrealistic values (>1M courses)

### API Behavior Validation

- Validates `pagingRequestId` consistency during pagination
- Detects empty responses when more data is expected
- Clear error messages for debugging

### Comprehensive Logging

Added detailed logging throughout pagination flow for monitoring and debugging.

## Expected Results

**API Call Reduction**: Maximum 100 calls per sync (enforced by safety limit)  
**Improved Reliability**: Connector completes with available data even if API has issues  
**Better Diagnostics**: Specific error messages identify root causes  
**Correct API Usage**: Each endpoint now uses its documented pagination pattern

## Monitoring

Key patterns to monitor in logs:

- High page counts (>50 pages per sync)
- Pagination safety limit triggers
- API behavior validation errors
- Sync frequency and duration

## Configuration

No configuration changes required. All changes are backwards compatible and include safety limits to prevent service disruption.
