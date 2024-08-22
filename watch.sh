#!/bin/bash
fswatch -0 -o content static templates | xargs -0 -n1 -I{} sh -c 'go build && ./debugjois.dev build --dev'
