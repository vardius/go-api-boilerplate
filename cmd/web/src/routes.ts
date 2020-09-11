const routesMap = {
  home: '/',
  login: '/login',
  client_authorize: '/authorize',
  security: '/security',
  // account_task_edit: makePath("/account/tasks/:id/edit")
};

const devRoutesMap = {};

const routes = Object.assign(
  routesMap,
  process.env.NODE_ENV !== "development" ? {} : devRoutesMap
)

const getPath = (path: string) => {
  // @ts-ignore
  const route = routes[path]
  if (!route) {
    throw new Error("invalid route");
  }

  return route
};

export default getPath;
