import * as React from "react";
import TextareaAutosize from "@mui/material/TextareaAutosize";

export default function LogView(props) {
  const { log } = props;
  return (
    <TextareaAutosize
      aria-label="empty textarea"
      placeholder="Log..."
      style={{ width: "100%" }}
      value={log}
    />
  );
}
