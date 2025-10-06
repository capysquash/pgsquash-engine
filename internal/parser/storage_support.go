package parser

import (
	"regexp"
	"strings"
)

// StoragePattern represents a recognized storage operation pattern
type StoragePattern struct {
	BucketName  string
	PolicyType  string
	Operation   string
	UserPattern string
	IsPublic    bool
}

// parseStorageOperation analyzes Supabase storage-related SQL statements
func parseStorageOperation(stmt *Statement) *StoragePattern {
	sql := strings.ToLower(stmt.SQL)

	// Storage bucket operations
	if strings.Contains(sql, "storage.buckets") {
		return parseStorageBucket(stmt)
	}

	// Storage policy operations
	if strings.Contains(sql, "storage.objects") {
		return parseStoragePolicy(stmt)
	}

	return nil
}

// parseStorageBucket extracts bucket configuration details
func parseStorageBucket(stmt *Statement) *StoragePattern {
	sql := stmt.SQL

	pattern := &StoragePattern{}

	// Extract bucket name
	bucketPattern := regexp.MustCompile(`(?i)(?:'|")([^'"]+)(?:'|").*?(?:'|")([^'"]+)(?:'|").*?(?:true|false)`)
	if matches := bucketPattern.FindStringSubmatch(sql); len(matches) >= 3 {
		pattern.BucketName = matches[1] // First string is usually the bucket ID
		pattern.IsPublic = strings.Contains(strings.ToLower(sql), "true")
	}

	// Extract additional bucket properties
	if strings.Contains(sql, "file_size_limit") {
		pattern.Operation = "CREATE_BUCKET_WITH_LIMITS"
	} else {
		pattern.Operation = "CREATE_BUCKET"
	}

	return pattern
}

// parseStoragePolicy extracts storage policy details
func parseStoragePolicy(stmt *Statement) *StoragePattern {
	sql := strings.ToLower(stmt.SQL)

	pattern := &StoragePattern{}

	// Extract bucket name from policy
	bucketPattern := regexp.MustCompile(`bucket_id\s*=\s*['"]([^'"]+)['"]`)
	if matches := bucketPattern.FindStringSubmatch(sql); len(matches) >= 2 {
		pattern.BucketName = matches[1]
	}

	// Determine policy type and operation
	if strings.Contains(sql, "for insert") {
		pattern.PolicyType = "INSERT"
		pattern.Operation = "UPLOAD_POLICY"
	} else if strings.Contains(sql, "for select") {
		pattern.PolicyType = "SELECT"
		pattern.Operation = "VIEW_POLICY"
	} else if strings.Contains(sql, "for delete") {
		pattern.PolicyType = "DELETE"
		pattern.Operation = "DELETE_POLICY"
	} else if strings.Contains(sql, "for update") {
		pattern.PolicyType = "UPDATE"
		pattern.Operation = "UPDATE_POLICY"
	}

	// Detect user access patterns
	pattern.UserPattern = detectUserAccessPattern(sql)

	return pattern
}

// detectUserAccessPattern identifies how users access storage objects
func detectUserAccessPattern(sql string) string {
	// JWT v2 patterns
	if strings.Contains(sql, "auth.jwt()->>'sub'") {
		return "JWT_V2_USER_FOLDER"
	}

	// Legacy auth patterns
	if strings.Contains(sql, "auth.uid()") {
		return "SUPABASE_USER_FOLDER"
	}

	// Clerk-specific patterns
	if strings.Contains(sql, "clerk_user_id()") {
		return "CLERK_USER_FOLDER"
	}

	// Folder-based access
	if strings.Contains(sql, "storage.foldername") {
		if strings.Contains(sql, "[1]") {
			return "USER_FOLDER_ACCESS"
		}
		return "FOLDER_BASED_ACCESS"
	}

	// Public access
	if strings.Contains(sql, "bucket_id =") && !strings.Contains(sql, "auth.") {
		return "PUBLIC_ACCESS"
	}

	// Organization-based access (JWT v2)
	if strings.Contains(sql, "auth.jwt()->'o'->") {
		return "ORGANIZATION_ACCESS"
	}

	return "UNKNOWN_PATTERN"
}

// Enhanced storage object tracking
func addStorageTracking(stmt *Statement) {
	if storagePattern := parseStorageOperation(stmt); storagePattern != nil {
		// Add storage-specific metadata
		if stmt.Comments == nil {
			stmt.Comments = make([]string, 0)
		}

		stmt.Comments = append(stmt.Comments,
			"Storage Pattern: "+storagePattern.Operation,
			"Bucket: "+storagePattern.BucketName,
			"User Pattern: "+storagePattern.UserPattern,
		)

		// Update dependencies to include bucket references
		if storagePattern.BucketName != "" && stmt.ObjectType == TypePolicy {
			stmt.Dependencies = append(stmt.Dependencies, "storage.buckets:"+storagePattern.BucketName)
		}
	}
}

// generateStoragePolicyName creates a consistent naming scheme for storage policies
func generateStoragePolicyName(bucketName, operation, userPattern string) string {
	parts := []string{bucketName, operation, userPattern}
	return strings.ToLower(strings.Join(parts, "_"))
}

// isStorageRelated checks if a statement involves Supabase storage
func isStorageRelated(stmt *Statement) bool {
	sql := strings.ToLower(stmt.SQL)
	return strings.Contains(sql, "storage.") ||
		strings.Contains(sql, "buckets") ||
		(stmt.ObjectType == TypePolicy && strings.Contains(sql, "bucket_id"))
}

// consolidateStoragePolicies groups related storage policies for optimization
func consolidateStoragePolicies(statements []*Statement) map[string][]*Statement {
	bucketPolicies := make(map[string][]*Statement)

	for _, stmt := range statements {
		if isStorageRelated(stmt) {
			if storagePattern := parseStorageOperation(stmt); storagePattern != nil {
				bucketName := storagePattern.BucketName
				if bucketName == "" {
					bucketName = "unknown"
				}

				bucketPolicies[bucketName] = append(bucketPolicies[bucketName], stmt)
			}
		}
	}

	return bucketPolicies
}

// Storage-specific validation patterns
var storageValidationPatterns = []struct {
	Name    string
	Pattern *regexp.Regexp
	Issue   string
}{
	{
		Name:    "MissingBucketReference",
		Pattern: regexp.MustCompile(`(?i)create\s+policy.*storage\.objects.*without.*bucket_id`),
		Issue:   "Storage policy without bucket_id constraint",
	},
	{
		Name:    "UnrestrictedStorageAccess",
		Pattern: regexp.MustCompile(`(?i)create\s+policy.*storage\.objects.*using\s*\(\s*true\s*\)`),
		Issue:   "Unrestricted storage access policy",
	},
	{
		Name:    "LegacyAuthPattern",
		Pattern: regexp.MustCompile(`(?i)auth\.uid\(\).*storage\.objects`),
		Issue:   "Legacy Supabase auth pattern in storage policy",
	},
}

// validateStoragePolicy checks storage policies for common issues
func validateStoragePolicy(stmt *Statement) []string {
	var issues []string

	for _, validation := range storageValidationPatterns {
		if validation.Pattern.MatchString(stmt.SQL) {
			issues = append(issues, validation.Issue)
		}
	}

	return issues
}
