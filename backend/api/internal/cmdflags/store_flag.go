package cmdflags

import "flag"

var _ = flag.String("store", "", "Store transcript JSON in the given S3 bucket ARN")
