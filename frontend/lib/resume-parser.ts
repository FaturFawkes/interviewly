const ALLOWED_EXTENSIONS = new Set(["pdf", "docx", "txt", "md", "rtf"]);

function getFileExtension(fileName: string): string {
  const normalized = fileName.trim().toLowerCase();
  const dotIndex = normalized.lastIndexOf(".");
  if (dotIndex < 0 || dotIndex === normalized.length - 1) {
    return "";
  }

  return normalized.slice(dotIndex + 1);
}

function normalizeExtractedText(value: string): string {
  return value
    .replace(/\u0000/g, " ")
    .replace(/\r/g, "\n")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}

function ensureSupportedFile(file: File): string {
  const extension = getFileExtension(file.name);
  if (!ALLOWED_EXTENSIONS.has(extension)) {
    throw new Error("Format file tidak didukung. Gunakan PDF, DOCX, TXT, MD, atau RTF.");
  }

  return extension;
}

async function extractPdfText(file: File): Promise<string> {
  const pdfjs = await import("pdfjs-dist/legacy/build/pdf.mjs");
  const workerSrc = new URL("pdfjs-dist/legacy/build/pdf.worker.min.mjs", import.meta.url).toString();

  if (pdfjs.GlobalWorkerOptions.workerSrc !== workerSrc) {
    pdfjs.GlobalWorkerOptions.workerSrc = workerSrc;
  }

  const bytes = new Uint8Array(await file.arrayBuffer());
  const loadingTask = pdfjs.getDocument({
    data: bytes,
  });
  const document = await loadingTask.promise;

  const pages: string[] = [];
  for (let pageNumber = 1; pageNumber <= document.numPages; pageNumber += 1) {
    const page = await document.getPage(pageNumber);
    const textContent = await page.getTextContent();
    const pageText = textContent.items
      .map((item) => {
        const candidate = item as { str?: string };
        return candidate.str ?? "";
      })
      .join(" ")
      .trim();

    if (pageText) {
      pages.push(pageText);
    }
  }

  return normalizeExtractedText(pages.join("\n\n"));
}

async function extractDocxText(file: File): Promise<string> {
  const mammoth = await import("mammoth");
  const result = await mammoth.extractRawText({
    arrayBuffer: await file.arrayBuffer(),
  });

  return normalizeExtractedText(result.value ?? "");
}

async function extractPlainText(file: File): Promise<string> {
  return normalizeExtractedText(await file.text());
}

export async function extractTextFromResumeFile(file: File): Promise<string> {
  const extension = ensureSupportedFile(file);

  if (extension === "pdf") {
    return extractPdfText(file);
  }

  if (extension === "docx") {
    return extractDocxText(file);
  }

  return extractPlainText(file);
}

export function getAllowedResumeExtensionsLabel(): string {
  return Array.from(ALLOWED_EXTENSIONS).map((ext) => `.${ext}`).join(", ");
}
