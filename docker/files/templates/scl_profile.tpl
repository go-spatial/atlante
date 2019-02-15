#!/bin/bash

# Enable the {{.package}} by default

source /opt/rh/{{.package}}/enable
export X_SCLS="`scl enable {{.package}} 'echo $X_SCLS'`"

