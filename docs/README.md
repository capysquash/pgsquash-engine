# pg-squash Documentation

PostgreSQL migration squasher and optimizer.

## Getting Started

- [**Quickstart**](quickstart.md) - Get running in 5 minutes
- [**CLI Reference**](cli-reference.md) - All commands and flags
- [**Configuration**](configuration.md) - Config file options

## Guides

- [**Safety Levels**](safety-levels.md) - Choosing the right safety mode
- [**AI Features**](ai-features.md) - Optional AI-powered analysis
- [**Troubleshooting**](troubleshooting.md) - Common issues and solutions

## Deployment

- [**Docker**](deployment/docker.md) - Container usage and validation
- [**GitHub Integration**](deployment/github-integration.md) - PR automation
- [**Production**](deployment/production.md) - Production deployment best practices

## Development

- [**Architecture**](architecture.md) - System design and internals
- [**Contributing**](dev/CONTRIBUTING.md) - How to contribute
- [**Development Setup**](dev/development.md) - Dev environment setup
- [**Testing**](dev/testing.md) - Test strategy and execution

## Quick Reference

### Common Commands

```bash
# Analysis
pgsquash analyze migrations/*.sql
pgsquash analyze-deep migrations/*.sql              # AI-powered

# Squashing
pgsquash squash migrations/*.sql --dry-run
pgsquash squash migrations/*.sql --output clean/

# Workflows
pgsquash safe migrations/*.sql                      # Production
pgsquash fast migrations/*.sql                      # Development

# Validation
pgsquash validate migrations/ clean/

# Configuration
pgsquash init-config
```

### Safety Levels

| Level | Environment | Optimization |
|-------|-------------|--------------|
| paranoid | Critical production | Minimal |
| conservative | Production | CREATE+ALTER only |
| standard | Staging | Balanced |
| aggressive | Development | Maximum |

### Key Config Options

```json
{
  "safety_level": "standard",
  "output": {
    "directory": "squashed"
  },
  "performance": {
    "streaming": true,
    "parallel_processing": true
  }
}
```

## Documentation by Role

### Database Administrator
1. [Quickstart](quickstart.md)
2. [Safety Levels](safety-levels.md)
3. [Production Guide](deployment/production.md)
4. [Troubleshooting](troubleshooting.md)

### Developer
1. [Quickstart](quickstart.md)
2. [CLI Reference](cli-reference.md)
3. [Configuration](configuration.md)
4. [Docker](deployment/docker.md)

### DevOps Engineer
1. [GitHub Integration](deployment/github-integration.md)
2. [Docker](deployment/docker.md)
3. [Configuration](configuration.md)
4. [Architecture](architecture.md)

### Contributor
1. [Architecture](architecture.md)
2. [Contributing](dev/CONTRIBUTING.md)
3. [Development Setup](dev/development.md)
4. [Testing](dev/testing.md)

## External Resources

- **PostgreSQL Docs**: https://www.postgresql.org/docs/
- **pg_query**: https://github.com/pganalyze/pg_query_go
- **Anthropic Claude**: https://docs.anthropic.com/
- **OpenAI**: https://platform.openai.com/docs/

---

**Version**: 0.8.1-beta (docs for 1.0.0 in progress)
**Last Updated**: 2025-10-06
