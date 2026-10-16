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
const topInput = document.getElementById("top");
const topPresets = document.getElementById("top-presets");

let objectURL = null;
let controller = null;

// Target-type segmented control reveals the relevant field group.
for (const radio of form.querySelectorAll('input[name="target_type"]')) {
  radio.addEventListener("change", () => {
    if (radio.checked) form.dataset.target = radio.value;
  });
}

// Top-N presets drive the hidden `top` field (0 = keep everything).
topPresets?.addEventListener("click", (event) => {
  const chip = event.target.closest(".chip");
  if (!chip) return;
  topInput.value = chip.dataset.top;
  for (const c of topPresets.querySelectorAll(".chip")) {
    c.classList.toggle("is-active", c === chip);
  }
});

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
    result.scrollIntoView({ block: "nearest" });
    return;
  }

  const hashing = form.querySelectorAll('input[name="hashes"]:checked').length > 0;
  const topN = Number(data.get("top"));

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
      result.scrollIntoView({ block: "nearest" });
      return;
    }

    const filename = hashing ? "passwords-hashes.txt" : "passwords.txt";
    link.download = filename;
    link.innerHTML = `Download ${filename} <span aria-hidden="true">↓</span>`;
    previewLabel.textContent = hashing
      ? "Preview — candidate : hash…"
      : "Preview first 20 candidates";
    preview.textContent = (await blob.slice(0, 32768).text())
      .trimEnd()
      .split("\n")
      .slice(0, 20)
      .join("\n");

    const count = Number(
      response.headers.get("X-Candidate-Count"),
    ).toLocaleString();
    let message =
      topN > 0
        ? `Top ${count} candidates (ranked best-first) ready.`
        : `${count} unique candidates ready. Download your wordlist below.`;
    if (hashing) message += " Each line pairs a candidate with its hash(es).";
    if (response.headers.get("X-Limit-Reached") === "true") {
      message +=
        " Candidate budget reached — raise it or export a CLI config to explore more.";
    }
    if (response.headers.get("X-Search-Limited") === "true") {
      message +=
        " Search limits were reached, so this is a partial list; export an exhaustive config for a complete run.";
    }
    statusText.textContent = message;
    download.hidden = false;
    result.scrollIntoView({ block: "nearest" });
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
