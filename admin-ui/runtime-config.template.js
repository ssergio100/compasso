(() => {
  const configuredAPIBaseURL = "${COMPASSO_API_BASE_URL}";
  const apiPort = "${COMPASSO_API_PORT}";
  const automaticAPIBaseURL = `${window.location.protocol}//${window.location.hostname}:${apiPort}`;
  window.COMPASSO_CONFIG = Object.freeze({
    apiBaseURL: configuredAPIBaseURL === "auto" ? automaticAPIBaseURL : configuredAPIBaseURL,
  });
})();
