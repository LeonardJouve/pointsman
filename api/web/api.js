export async function request(url, payload) {
  const options = payload === undefined ? undefined : {method: "POST", body: JSON.stringify(payload)};
  const response = await fetch(url, options);
  const body = await response.json();

  if (!response.ok) {
    return [undefined, body.error];
  }

  return [body, undefined];
}
