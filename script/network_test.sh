#!/usr/bin/env
set -u

RESULT="baseline"
mkdir -p "../docs/$RESULT"

REPORT="docs/$RESULT/report.txt"
FAILED=0
GO_PATH="$(where.exe go 2>/dev/null | head -n 1 | tr -d '\r')"
RG_PATH="$(where.exe rg 2>/dev/null | head -n 1 | tr -d '\r')"

run_check() {
     local name="$1"
     shift

     echo 
     echo "==================================================="
     echo "Running: $name"
     echo "==================================================="
     {
          echo 
          echo "==================================================="
          echo "Running: $name"
          echo "==================================================="
     } >> "$REPORT"

     "$@" 2>&1 | tee -a "$REPORT"
     local status=${PIPESTATUS[0]}

     if ["$status" -ne 0]; then
          echo "FAILED (exit code: $status)"
          echo "FAILED (exit code: $status)" >> "$REPORT" 
          FAILED=1
     else
          echo "PASSED"
          echo "PASSED" >> "$REPORT"
     fi
}

if [ -z "$GO_PATH" ];
     then 
          echo "ERROR: go is not installed or not in path."
          exit 1
fi

if [ -z "$RG_PATH" ];
     then 
          echo "ERROR: rg is not installed or not in path."
          exit 1
fi

{
     echo "Leoxy Baseline"
     echo "Date: $(date)"
     echo "GO version: $(go version)"
     echo "GOOS: $(go env GOOS) - GOARCH: $(go env GOARCH)"
     echo
     echo "Git status:"
     git status --short 2>&1 || true
} > "$REPORT"

run_check "GO test" go test ./...
run_check "GO vet" go vet ./...
run_check "GO detector" go test-race ./...

run_check "Network listeners and servers" \
     rg -n '(ListenAndServe|http\.Serve|net\.Listen|ListenPacket|ListenUDP|Accept\(|AcceptUDP|Serve\()' .

run_check "HTTP Proxy and handlers" \
     rg -n 'ReverseProxy|RoundTrip|Transport|ServeHTTP|http\ .Hander' .

run_check "Data transfer and connectins" \
     rg -n '.io\.Copy|ReadFrom|WriteTo|DialContext|SetDeadLine|Close\(' .

run_check "Existing benchmarks" \
     go test ./... -run '^$' -bench . -benchmem

echo 
echo "==================================================="
echo "Baseline finished"
echo "Report Name: $REPORT"
echo "==================================================="

if [ "$FAILED" -ne 0]; then
     echo "Same checks failed or returned non-zero status. Review the report"
     exit 1
if

echo "All Check Success"
