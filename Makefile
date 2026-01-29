.PHONY: view capture help

help:
	@echo "Usage:"
	@echo "  make view              - Start the web viewer (http://localhost:8000)"
	@echo "  make capture name=...  - Capture a new Claude session"
	@echo "                           Example: make capture name=test-session"

view:
	./bin/viewer

# Helper to run capture via make
capture:
	@if [ -z "$(name)" ]; then \
		echo "Error: 'name' is required. Example: make capture name=my-session"; \
		exit 1; \
	fi
	./bin/claude-capture $(name)
