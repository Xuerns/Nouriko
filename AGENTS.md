## Rule For Agents
# Always Validate
- Run all validation in server folder if you changed server code
- Run 'go vet ./...' on server folder
- Run 'gosec ./...' on server folder
- Run 'go test ./...' on server folder
- Run 'go fmt ./...' on server folder
- Run 'go build ./...' on server folder

- Run all validation in client folder if you changed client code
- Run 'pnpm lint' on client folder
- Run 'pnpm run check' on client folder
- Run 'pnpm run format' on client folder
- Run 'pnpm run build' on client folder

# Client Rules
- Always use **pnpm** as the package manager
- Always use **TypeScript** as the language
- Always use **Tailwind CSS** as the styling
- Never use emoji in the code
- Always make the code clean and readable
- Always follow the **Next.js** guidelines
- Never use shadow drop
- Never use pulse animation
- Makesure the library save

# Important Notes
- Never hardcode any API keys and creditials, use environment variables (.env) instead
- Never manipulate the test just to make it PASS, if test failed fix the logic not the test
- Read the DESIGN.md, you can combine the DESIGN.md color to component shacdcn, react bits, and etc

# Important for components
- NEVER build standard UI primitives (Button, Input, Dialog/Modal, Select, Dropdown, Tabs, Card, Table, Toast, Accordion, Tooltip, etc.) or common animated UI effects from scratch.
- Follow this strict order whenever a UI component is needed:
  1. **Reuse first:** Check existing components in `frontend/src/components/ui` (or `frontend/components/ui`) before adding anything new.
  2. **Install from shadcn/ui:** If the component is not installed yet, run `pnpm dlx shadcn@latest add <component-name> --yes` inside the frontend folder.
  3. **Install from React Bits:** For animated/interactive UI components, check React Bits first and install the **TypeScript + Tailwind CSS (TS-TW)** variant via CLI (e.g., `pnpm dlx shadcn@latest add https://reactbits.dev/r/<ComponentName>-TS-TW --yes` or `pnpm dlx jsrepo add ...`) instead of manually writing animation code.
  4. **Custom as last resort:** Only create a custom component if it does not exist in shadcn/ui or React Bits. Compose it using existing shadcn/ui primitives and explain briefly why a custom component is necessary.
- Always run CLI commands with non-interactive flags (`--yes` / `-y`) and never manually copy-paste or recreate shadcn/ui internals by hand.
- Keep installed library components intact; customize styling via Tailwind classes or wrapper components instead of rewriting base files.

# Documentation you can refer
- Next.js: https://nextjs.org/docs
- Tailwind: https://tailwindcss.com/docs/
- Lucide: https://lucide.dev/icons/
