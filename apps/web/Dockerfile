FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# Public env is embedded at build time by Astro/Vite.
ARG PUBLIC_API_URL=http://localhost:8080
ARG PUBLIC_USE_MOCKS=false
ENV PUBLIC_API_URL=$PUBLIC_API_URL \
    PUBLIC_USE_MOCKS=$PUBLIC_USE_MOCKS
RUN npm run build

FROM node:22-alpine AS runtime
ENV NODE_ENV=production
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev && chown -R app:app /app
COPY --from=build --chown=app:app /app/dist ./dist
EXPOSE 4321
ENV HOST=0.0.0.0 PORT=4321
USER app
CMD ["node", "./dist/server/entry.mjs"]
