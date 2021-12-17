import * as React from "react";
import Box from "@mui/material/Box";
import List from "@mui/material/List";
import TextField from "@mui/material/TextField";

import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemText from "@mui/material/ListItemText";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import { VariableSizeList } from "react-window";
import { useState, useEffect } from "react";
import { docall } from "./utils.js";

function renderRow(props) {
  const { index, style, data } = props;

  return (
    <ListItem style={style} key={index} component="div" disablePadding>
      <ListItemButton>
        <ListItemText
          primary={`step[${index}]\tPC:${data.traces[index].pc}\t${data.traces[index].op}`}
        />
        {/* <ListItemText primary={`${data[index].op}`} /> */}
      </ListItemButton>
    </ListItem>
  );
}

export default function DebugView() {
  const [fullTrace, setFullTrace] = useState({});
  const [traceID, setTraceID] = useState({});
  const searchParams = new URLSearchParams(window.location.search);
  const txhash = searchParams.get("txhash");
  console.log(txhash);

  const listRef = React.createRef();

  useEffect(() => {
    docall("debug_traceTransaction", [txhash])
      .then((res) => res.json())
      .then(
        (result) => {
          if (result.hasOwnProperty("result")) {
            const traceResult = result.result;
            setFullTrace({
              traces: traceResult["structLogs"],
              clicked: (x) => {
                console.log(x);
                setTraceID(x);
              },
              selected: (x) => {
                console.log(x, traceID);
                return x === traceID;
              },
            });
          }
          //   } else {
          //     setFullTrace(JSON.stringify({ err: "Trace not found" }));
          //   }
        },
        (error) => {
          console.log(error);
        }
      );
  }, []);
  return (
    <Box
      sx={{
        width: "100%",
        height: 400,
        maxWidth: 360,
        bgcolor: "background.paper",
      }}
    >
      <Box
        sx={{
          display: "flex",
          flexDirection: "column",
          alignItems: "left",
          "& > *": {
            m: 1,
          },
        }}
      >
        <ButtonGroup variant="outlined" aria-label="outlined button group">
          <Button
            onClick={() => {
              console.log(listRef.current);
              listRef.current.scrollToItem(2);
            }}
          >
            🔙
          </Button>
          <Button>➡️</Button>
        </ButtonGroup>
      </Box>
      <VariableSizeList
        ref={listRef}
        height={400}
        width={360}
        itemSize={(x) => 50}
        itemCount={fullTrace.traces ? fullTrace.traces.length : 0}
        itemData={fullTrace}
        // style={{ textAlign: "left" }}
      >
        {renderRow}
      </VariableSizeList>

      <TextField></TextField>
    </Box>
  );
}
