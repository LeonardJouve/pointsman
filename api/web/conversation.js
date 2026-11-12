const bubbles = {
  user: (message) => bubble("bg-primary-subtle tail-left align-self-start", text(message)),
  assistant: (message) => bubble("bg-secondary-subtle tail-right align-self-end", text(message), ...calls(message)),
  tool: (message) => bubble("bg-warning-subtle tail-left align-self-start", block("pre", message.content)),
};

export function conversation(element, id, onStatus) {
  const column = document.createElement("div");
  column.className = "d-flex flex-column gap-3 w-75 mx-auto pt-3";
  element.replaceChildren(column);

  const source = new EventSource(`/sessions/${id}/events`);

  source.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    column.append(bubbles[message.role](message));
  });

  source.addEventListener("status", (event) => {
    source.close();
    onStatus(JSON.parse(event.data).status);
  });

  return source;
}

function bubble(classes, ...children) {
  const item = document.createElement("div");
  item.className = `position-relative p-2 rounded mw-100 text-break ${classes}`;
  item.append(...children.filter(Boolean));

  return item;
}

function text(message) {
  return message.content ? block("p", message.content, "mb-0") : null;
}

function calls(message) {
  return (message.tool_calls ?? []).map((call) => {
    const code = block("code", `${call.function.name}(${call.function.arguments})`);
    const pre = document.createElement("pre");
    pre.className = "mt-2 mb-0";
    pre.append(code);

    return pre;
  });
}

function block(tag, content, classes = "") {
  const element = document.createElement(tag);
  element.className = classes;
  element.textContent = content;

  return element;
}
