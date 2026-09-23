const form = document.getElementById("form"),
  button = document.getElementById("generate"),
  result = document.getElementById("result"),
  statusText = document.getElementById("status"),
  download = document.getElementById("download");
let objectURL;
let controller;
const cancel = document.getElementById("cancel");
cancel.addEventListener("click", () => controller?.abort());
form.addEventListener("submit", async (event) => {
  event.preventDefault();
  result.hidden = false;
  result.classList.remove("error");
  download.hidden = true;
  if (objectURL) {
    URL.revokeObjectURL(objectURL);
    objectURL = null;
  }
  const exporting = event.submitter?.id === "export-config";
  const data = new FormData(form);
  if (Number(data.get("min")) > Number(data.get("max"))) {
    result.classList.add("error");
    statusText.textContent = "Minimum length cannot exceed maximum length.";
    return;
  }
  controller = new AbortController();
  cancel.hidden = false;
  button.disabled = true;
  document.getElementById("export-config").disabled = true;
  button.textContent = "Generating…";
  statusText.textContent =
    "Processing your sources and checking the password policy…";
  try {
    const response = await fetch(exporting ? "/api/config" : "/api/generate", {
      method: "POST",
      headers: { "X-Pwprofiler": "local" },
      body: data,
      signal: controller.signal,
    });
    if (!response.ok) throw new Error(await response.text());
    const blob = await response.blob();
    objectURL = URL.createObjectURL(blob);
    document.getElementById("preview-label").textContent = exporting
      ? "Preview configuration"
      : "Preview first 20 candidates";
    document.getElementById("link").href = objectURL;
    document.getElementById("link").download = exporting
      ? "audit.yaml"
      : "passwords.txt";
    document.getElementById("link").textContent = exporting
      ? "Download audit.yaml ↓"
      : "Download passwords.txt ↓";
    if (exporting) {
      document.getElementById("preview").textContent = await blob.text();
      statusText.textContent =
        "Exhaustive configuration ready. Save audit.yaml, then run: pwprofiler generate -c audit.yaml -o passwords.txt. This configuration includes your input words; keep it private.";
      download.hidden = false;
      return;
    }
    document.getElementById("preview").textContent = (
      await blob.slice(0, 32768).text()
    )
      .trimEnd()
      .split("\n")
      .slice(0, 20)
      .join("\n");
    statusText.textContent =
      Number(response.headers.get("X-Candidate-Count")).toLocaleString() +
      " unique candidates ready. Download your wordlist below." +
      (response.headers.get("X-Limit-Reached") === "true"
        ? " Candidate limit reached; increase it to explore more results."
        : "") +
      (response.headers.get("X-Search-Limited") === "true"
        ? " Search limits were reached; this is a partial wordlist. See the algorithm guide for advanced configuration."
        : "");
    download.hidden = false;
  } catch (error) {
    result.classList.add("error");
    statusText.textContent =
      error.name === "AbortError"
        ? "Generation cancelled. Your inputs are ready to edit."
        : error.message ||
          "Generation failed. Check that the local server is running.";
  } finally {
    cancel.hidden = true;
    controller = null;
    button.disabled = false;
    document.getElementById("export-config").disabled = false;
    button.textContent = "Generate wordlist →";
  }
});
