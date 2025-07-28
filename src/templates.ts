import { readFileSync } from "fs";
import { type Note } from "./types";

export function formatContent(content: string): string {
  if (!content) {
    return "Empty note...";
  }

  // Escape HTML first
  let s = content
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

  // Apply basic markdown formatting

  // Code blocks (must be processed before line breaks)
  const codeBlockRegex = /```([\s\S]*?)```/g;
  const codeBlocks: string[] = [];
  const codeBlockPlaceholders: string[] = [];

  // Replace code blocks with placeholders to protect them from other formatting
  let match;
  let index = 0;
  while ((match = codeBlockRegex.exec(s)) !== null) {
    const placeholder = `__CODEBLOCK_${index}__`;
    codeBlocks.push(match[0]);
    const codeContent = match[1] ? match[1].trim() : "";
    codeBlockPlaceholders.push(`<pre><code>${codeContent}</code></pre>`);
    s = s.replace(match[0], placeholder);
    index++;
  }

  // Convert line breaks
  s = s.replace(/\n/g, "<br>");

  // Bold - fix the regex to be more specific
  s = s.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");

  // Italic - make sure it doesn't conflict with bold
  s = s.replace(/\*([^*\n]+)\*/g, "<em>$1</em>");

  // Inline code (process after code blocks to avoid conflicts)
  s = s.replace(/`([^`\n]+)`/g, "<code>$1</code>");

  // Simple heading support
  s = s.replace(
    /^# (.+)$/gm,
    "<strong style='font-size:1.1em;color:#333;'>$1</strong>"
  );

  // Restore code blocks
  codeBlockPlaceholders.forEach((processedBlock, i) => {
    s = s.replace(`__CODEBLOCK_${i}__`, processedBlock);
  });

  return s;
}

export function renderTemplate(templatePath: string, data: any): string {
  let template = readFileSync(templatePath, "utf-8");

  // For index.html, we need special handling due to complex nested templates
  if (templatePath.includes("index.html")) {
    return renderIndexTemplate(template, data);
  }

  // Simple template engine for other templates
  return renderSimpleTemplate(template, data);
}

function renderIndexTemplate(template: string, data: any): string {
  // Step 1: Handle len function
  template = template.replace(/\{\{len \.([^}]+)\}\}/g, (match, prop) => {
    const value = data[prop];
    return Array.isArray(value) ? String(value.length) : "0";
  });

  // Step 2: Handle the main if-else-end block for Notes FIRST (before any property replacement)
  const notesIfElsePattern =
    /\{\{if \.Notes\}\}([\s\S]*?)\{\{else\}\}([\s\S]*?)\{\{end\}\}/;
  template = template.replace(
    notesIfElsePattern,
    (match, notesContent, emptyContent) => {
      if (data.Notes && data.Notes.length > 0) {
        // Handle range block
        let processedNotesContent = notesContent.replace(
          /\{\{range \.Notes\}\}([\s\S]*?)\{\{end\}\}/,
          (_rangeMatch: string, noteTemplate: string) => {
            return data.Notes.map((note: Note) => {
              let processedTemplate = noteTemplate;

              // Replace note properties
              processedTemplate = processedTemplate.replace(
                /\{\{\.ID\}\}/g,
                note.id
              );
              processedTemplate = processedTemplate.replace(
                /\{\{\.Content\}\}/g,
                note.content
              );

              // Handle formatContent pipe
              processedTemplate = processedTemplate.replace(
                /\{\{\.Content \| formatContent\}\}/g,
                formatContent(note.content)
              );

              return processedTemplate;
            }).join("");
          }
        );

        return processedNotesContent;
      } else {
        // Handle nested if-else in empty state
        let processedEmptyContent = emptyContent.replace(
          /\{\{if \.Query\}\}([\s\S]*?)\{\{else\}\}([\s\S]*?)\{\{end\}\}/,
          (
            _emptyMatch: string,
            queryContent: string,
            noQueryContent: string
          ) => {
            if (data.Query && data.Query.trim() !== "") {
              return queryContent.replace(/\{\{\.Query\}\}/g, data.Query);
            } else {
              return noQueryContent;
            }
          }
        );
        return processedEmptyContent;
      }
    }
  );

  // Step 3: Replace simple property references (only for top-level data, not note properties)
  template = template.replace(/\{\{\.Query\}\}/g, () => {
    return data.Query || "";
  });

  // Step 4: Handle remaining simple if blocks ({{if .Query}} type, but NOT {{if .Notes}})
  let ifMatches;
  let iteration = 0;
  while (
    (ifMatches = template.match(/\{\{if \.([^}]+)\}\}/)) &&
    iteration < 10
  ) {
    iteration++;
    const prop = ifMatches[1];
    if (!prop) continue;

    // Skip if this is a Notes block (should have been handled above)
    if (prop === "Notes") break;

    const value = data[prop];
    const shouldShow =
      value && (typeof value === "string" ? value.trim() !== "" : true);

    // Find the matching {{end}} for this {{if}}
    const ifStart = template.indexOf(ifMatches[0]);
    const searchStart = ifStart + ifMatches[0].length;

    let endPos = -1;
    let depth = 1;
    let pos = searchStart;

    while (depth > 0 && pos < template.length) {
      const nextIf = template.indexOf("{{if", pos);
      const nextElse = template.indexOf("{{else}}", pos);
      const nextEnd = template.indexOf("{{end}}", pos);

      let nextAction = Math.min(
        nextIf === -1 ? Infinity : nextIf,
        nextElse === -1 ? Infinity : nextElse,
        nextEnd === -1 ? Infinity : nextEnd
      );

      if (nextAction === Infinity) break;

      if (nextAction === nextIf) {
        depth++;
        pos = nextIf + 5;
      } else if (nextAction === nextEnd) {
        depth--;
        if (depth === 0) {
          endPos = nextEnd;
        }
        pos = nextEnd + 7;
      } else {
        // nextElse
        pos = nextElse + 8;
      }
    }

    if (endPos !== -1) {
      const content = template.substring(searchStart, endPos);
      const replacement = shouldShow ? content : "";
      template =
        template.substring(0, ifStart) +
        replacement +
        template.substring(endPos + 7);
    } else {
      break;
    }
  }

  // Step 5: Clean up any remaining orphaned {{end}} tags
  template = template.replace(/\{\{end\}\}/g, "");

  return template;
}

function renderSimpleTemplate(template: string, data: any): string {
  // Handle len function first
  template = template.replace(/\{\{len \.([^}]+)\}\}/g, (match, prop) => {
    const value = data[prop];
    if (Array.isArray(value)) {
      return String(value.length);
    }
    return "0";
  });

  // Handle if-else blocks
  template = template.replace(
    /\{\{if \.([^}]+)\}\}([\s\S]*?)\{\{else\}\}([\s\S]*?)\{\{end\}\}/g,
    (match, prop, ifContent, elseContent) => {
      return data[prop] ? ifContent : elseContent;
    }
  );

  // Handle simple if blocks (without else)
  template = template.replace(
    /\{\{if \.([^}]+)\}\}([\s\S]*?)\{\{end\}\}/g,
    (match, prop, content) => {
      return data[prop] ? content : "";
    }
  );

  // Handle simple property replacements
  template = template.replace(/\{\{\.([^}|\s]+)\}\}/g, (match, prop) => {
    const value = data[prop];
    return value !== undefined && value !== null ? String(value) : "";
  });

  return template;
}
