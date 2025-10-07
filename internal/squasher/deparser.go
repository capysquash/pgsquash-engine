package squasher

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// Deparse takes a modified pg_query.ParseResult and generates a SQL string.
// This is the primary interface for converting AST back to SQL.
func Deparse(tree *pg_query.ParseResult) (string, error) {
	if tree == nil {
		return "", nil
	}

	res, err := pg_query.Deparse(tree)
	if err != nil {
		return "", fmt.Errorf("failed to deparse tree: %w", err)
	}
	return res, nil
}

//nolint:unused // Utility function for future node deparsing needs
// deparseNode converts a single Node back to SQL using pg_query.Deparse
func deparseNode(node *pg_query.Node) (string, error) {
	if node == nil {
		return "", nil
	}
	// Create a dummy ParseResult to use the real deparser
	tree := &pg_query.ParseResult{
		Stmts: []*pg_query.RawStmt{
			{
				Stmt: node,
			},
		},
	}
	return Deparse(tree)
}
