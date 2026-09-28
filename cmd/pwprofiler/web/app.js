const form = document.getElementById("form");
const button = document.getElementById("generate");
const result = document.getElementById("result");
const statusText = document.getElementById("status");
const download = document.getElementById("download");
const cancel = document.getElementById("cancel");
const exportButton = document.getElementById("export-config");
const link = document.getElementById("link");
const preview = document.getElementById("preview");
const previewLabel = document.getElementById("preview-label");
const copyButton = document.getElementById("copy");

let objectURL = null;
let controller = null;

// Show how many files each upload holds, and mark the card as filled.
for (const input of document.querySelectorAll('input[type="file"]')) {
  const badge = input.parentElement.querySelector(".upload-count");
  if (!badge) continue;
  input.addEventListener("change", () => {
    const n = input.files.length;
    if (n === 0) {
      badge.textContent = badge.dataset.empty;
      badge.classList.remove("has-files");
    } else {
      badge.textContent = n === 1 ? input.files[0].name : `${n} files selected`;
      badge.classList.add("has-files");
    }
  });
}

cancel.addEventListener("click", () => controller?.abort());

copyButton?.addEventListener("click", async () => {
  try {
    await navigator.clipboard.writeText(preview.textContent);
    const original = copyButton.textContent;
    copyButton.textContent = "Copied ✓";
    setTimeout(() => (copyButton.textContent = original), 1500);
  } catch {
    copyButton.textContent = "Copy failed";
  }
});

function revoke() {
  if (objectURL) {
    URL.revokeObjectURL(objectURL);
    objectURL = null;
  }
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const exporting = event.submitter?.id === "export-config";

  result.hidden = false;
  result.classList.remove("error");
  download.hidden = true;
  revoke();

  const data = new FormData(form);
  if (Number(data.get("min")) > Number(data.get("max"))) {
    result.classList.add("error");
    statusText.textContent = "Minimum length cannot exceed maximum length.";
    return;
  }

  controller = new AbortController();
  cancel.hidden = false;
  button.disabled = true;
  exportButton.disabled = true;
  result.classList.add("loading");
  button.textContent = "Generating…";
  statusText.textContent = exporting
    ? "Building an exhaustive CLI configuration…"
    : "Processing your sources and checking the password policy…";

  try {
    const response = await fetch(exporting ? "/api/config" : "/api/generate", {
      method: "POST",
      headers: { "X-Pwprofiler": "local" },
      body: data,
      signal: controller.signal,
    });
    if (!response.ok) throw new Error((await response.text()).trim());

    const blob = await response.blob();
    objectURL = URL.createObjectURL(blob);
    link.href = objectURL;

    if (exporting) {
      link.download = "audit.yaml";
      link.innerHTML = 'Download audit.yaml <span aria-hidden="true">↓</span>';
      previewLabel.textContent = "Preview configuration";
      preview.textContent = await blob.text();
      statusText.textContent =
        "Exhaustive configuration ready. Save audit.yaml, then run: pwprofiler generate -c audit.yaml -o passwords.txt — this file includes your input words, so keep it private.";
      download.hidden = false;
      return;
    }

    link.download = "passwords.txt";
    link.innerHTML =
      'Download passwords.txt <span aria-hidden="true">↓</span>';
    previewLabel.textContent = "Preview first 20 candidates";
    preview.textContent = (await blob.slice(0, 32768).text())
      .trimEnd()
      .split("\n")
      .slice(0, 20)
      .join("\n");

    const count = Number(
      response.headers.get("X-Candidate-Count"),
    ).toLocaleString();
    let message = `${count} unique candidates ready. Download your wordlist below.`;
    if (response.headers.get("X-Limit-Reached") === "true") {
      message +=
        " Candidate limit reached — increase it to explore more results.";
    }
    if (response.headers.get("X-Search-Limited") === "true") {
      message +=
        " Search limits were reached, so this is a partial list. See the algorithm guide for exhaustive runs.";
    }
    statusText.textContent = message;
    download.hidden = false;
  } catch (error) {
    result.classList.add("error");
    statusText.textContent =
      error.name === "AbortError"
        ? "Generation cancelled. Your inputs are ready to edit."
        : error.message ||
          "Generation failed. Check that the local server is running.";
  } finally {
    result.classList.remove("loading");
    cancel.hidden = true;
    controller = null;
    button.disabled = false;
    exportButton.disabled = false;
    button.innerHTML = 'Generate wordlist <span aria-hidden="true">→</span>';
  }
});
