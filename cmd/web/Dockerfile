FROM node:14.7.0 AS buildenv

LABEL maintainer="Rafał Lorenz <vardius@gmail.com>"

ARG BIN

ENV BIN=${BIN}
ENV NODE_ENV=production

# Create a location in the container for the source code.
RUN mkdir -p /app

# Copy the module files first and then download the dependencies. If this
# doesn't change, we won't need to do this again in future builds.
COPY cmd/"$BIN"/package.json cmd/"$BIN"/yarn.lock /app/

WORKDIR /app
RUN yarn

COPY cmd/"$BIN" ./
RUN yarn build

FROM node:14.7.0-alpine
COPY --from=buildenv /app/build /app

ENV PATH /node_modules/.bin:$PATH

RUN yarn global add serve

CMD ["serve", "-s", "app", "-l", "3000"]
