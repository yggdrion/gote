#!/usr/bin/env bun

import { renderTemplate } from "../src/templates";

const data = {
  Error: "",
  IsFirstTime: false,
};

try {
  const html = renderTemplate("./static/login.html", data);
  console.log(html);
} catch (err) {
  console.error("Error:", err);
}
