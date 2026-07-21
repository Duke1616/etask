package codeassist

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Duke1616/etask/internal/domain"
)

const validationTimeout = 5 * time.Second

// validateCandidate 执行与生成场景无关的语法和运行契约检查。
func validateCandidate(ctx context.Context, language, code string) []domain.AIDiagnostic {
	diagnostics := make([]domain.AIDiagnostic, 0)
	validationCtx, cancel := context.WithTimeout(ctx, validationTimeout)
	defer cancel()

	var command *exec.Cmd
	switch language {
	case "python":
		command = exec.CommandContext(validationCtx, "python3", "-c",
			"import ast,sys; ast.parse(sys.stdin.read())")
	case "shell":
		command = exec.CommandContext(validationCtx, "/bin/bash", "-n")
	default:
		return append(diagnostics, domain.AIDiagnostic{
			Severity: domain.AIDiagnosticSeverityError, Code: "UNSUPPORTED_LANGUAGE",
			Message: fmt.Sprintf("Unsupported script language: %s", language),
		})
	}
	command.Stdin = strings.NewReader(code)
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		diagnostics = append(diagnostics, domain.AIDiagnostic{
			Severity: domain.AIDiagnosticSeverityError, Code: "SYNTAX_ERROR", Message: message,
		})
	}

	legacyPatterns := []struct {
		code, pattern, message string
	}{
		{"LEGACY_POSITIONAL_ARGS", "sys.argv[1]", "Use ETASK_ARGS_FILE instead of sys.argv[1]."},
		{"LEGACY_POSITIONAL_VARIABLES", "sys.argv[2]", "Use ETASK_VARIABLES_FILE instead of sys.argv[2]."},
		{"LEGACY_SYSTEM_IMPORT", "from third_party", "Use the etask namespace for SYSTEM Python imports."},
		{"LEGACY_SHELL_SOURCE", "source ./third_party", "Use ETASK_SYSTEM_ROOT for SYSTEM Shell files."},
	}
	for _, pattern := range legacyPatterns {
		if strings.Contains(code, pattern.pattern) {
			diagnostics = append(diagnostics, domain.AIDiagnostic{
				Severity: domain.AIDiagnosticSeverityWarning,
				Code:     pattern.code, Message: pattern.message,
			})
		}
	}
	return diagnostics
}

func hasDiagnosticErrors(diagnostics []domain.AIDiagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == domain.AIDiagnosticSeverityError {
			return true
		}
	}
	return false
}
