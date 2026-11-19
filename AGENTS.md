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
- Always follow the **SvelteKit** guidelines
- Never use shadow drop
- Never use pulse animation
- Makesure the library save

# Important Notes
- Never hardcode any API keys and creditials, use environment variables (.env) instead
- Never manipulate the test just to make it PASS, if test failed fix the logic not the test

# Documentation you can refer
- Next.js: https://nextjs.org/docs
- Tailwind: https://tailwindcss.com/docs/
- Lucide: https://lucide.dev/icons/