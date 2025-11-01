# internal/plugins/auth package map

## Domain Summary
- Generates compatibility SQL blocks for various auth providers so validation environments mimic hosted behavior (Clerk, Supabase, Auth0, NextAuth, Firebase).
- Used by higher-level plugins to inject pre/post migration scaffolding.

## Files (alphabetical)

### compatibility.go
- **Purpose**: Service-specific compatibility generators and helpers.
- **Key Types**
  - `ServiceType`: Enum of supported providers (`Clerk`, `Supabase`, `Auth0`, `NextAuth`, `Firebase`, etc.).
  - `CompatibilityGenerator`: Holds service selection and generates SQL bundles.
- **Functions / Methods**
  - `NewCompatibilityGenerator`: Constructs generator from `ServiceType`.
  - `(*CompatibilityGenerator) Generate`: Delegates to service-specific generator.
  - Service generators: `GenerateClerkCompatibility`, `GenerateSupabaseCompatibility`, `GenerateAuth0Compatibility`, `GenerateNextAuthCompatibility`, `GenerateFirebaseCompatibility`.
  - Utility: `GetServiceFromString` (string → enum mapping).

## Subdirectories
- _None._
