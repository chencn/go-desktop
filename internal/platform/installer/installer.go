package installer

import "context"

type Runner func(ctx context.Context, installerPath string) error
