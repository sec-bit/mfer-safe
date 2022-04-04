import * as React from "react";
import { useState, useEffect } from "react";
import { styled, alpha } from "@mui/material/styles";
import AppBar from "@mui/material/AppBar";
import Box from "@mui/material/Box";
import Toolbar from "@mui/material/Toolbar";
import IconButton from "@mui/material/IconButton";
// import Typography from "@mui/material/Typography";
import InputBase from "@mui/material/InputBase";
import TextField from "@mui/material/TextField";
// import Badge from "@mui/material/Badge";
// import MenuItem from "@mui/material/MenuItem";
// import Menu from "@mui/material/Menu";
// import MenuIcon from "@mui/icons-material/Menu";
import SearchIcon from "@mui/icons-material/Search";
// import AccountCircle from "@mui/icons-material/AccountCircle";
// import MailIcon from "@mui/icons-material/Mail";
import ArrowBack from "@mui/icons-material/ArrowBack";
import ArrowForward from "@mui/icons-material/ArrowForward";
import Refresh from "@mui/icons-material/Refresh";
import Surfing from "@mui/icons-material/Surfing";
// import MoreIcon from "@mui/icons-material/MoreVert";

const Search = styled("div")(({ theme }) => ({
  position: "relative",
  borderRadius: theme.shape.borderRadius,
  backgroundColor: alpha(theme.palette.common.white, 0.15),
  "&:hover": {
    backgroundColor: alpha(theme.palette.common.white, 0.25),
  },
  marginRight: theme.spacing(2),
  marginLeft: 0,
  width: "100%",
  [theme.breakpoints.up("sm")]: {
    marginLeft: theme.spacing(3),
    width: "auto",
  },
}));

const SearchIconWrapper = styled("div")(({ theme }) => ({
  padding: theme.spacing(0, 2),
  height: "100%",
  position: "absolute",
  pointerEvents: "none",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
}));

const StyledInputBase = styled(InputBase)(({ theme }) => ({
  color: "inherit",
  "& .MuiInputBase-input": {
    padding: theme.spacing(1, 1, 1, 0),
    // vertical padding + font size from searchIcon
    paddingLeft: `calc(1em + ${theme.spacing(4)})`,
    transition: theme.transitions.create("width"),
    width: "100%",
    [theme.breakpoints.up("md")]: {
      width: "50ch",
    },
  },
}));

export default function NavigationBar() {
  const [target, setTarget] = useState("");
  const [stdout, setStdout] = useState("");

  const handleTargetInput = (e) => {
    setTarget(e.target.value);
  };

  useEffect(() => {
    if (window.api === undefined) {
      window.api = { receive: () => {} };
    }
    window.api.receive("target", (data) => {
      console.log(`Received ${data} from main process`);
      setTarget(data);
    });
    window.api.receive("stdout", (data) => {
      console.log(`Received ${data} from main process`);
      setStdout(data);
    });
  },[]);

  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar position="static">
        <Toolbar>
          <IconButton
            size="middle"
            edge="start"
            color="inherit"
            sx={{ mr: 2 }}
            onClick={() => {
              window.navigate.navigate({ action: "backward" });
            }}
          >
            <ArrowBack />
          </IconButton>
          <IconButton
            size="middle"
            edge="start"
            color="inherit"
            sx={{ mr: 2 }}
            onClick={() => {
              window.navigate.navigate({ action: "forward" });
            }}
          >
            <ArrowForward />
          </IconButton>
          <IconButton
            size="middle"
            edge="start"
            color="inherit"
            sx={{ mr: 0 }}
            onClick={() => {
              window.navigate.navigate({ action: "refresh" });
            }}
          >
            <Refresh />
          </IconButton>
          <Search>
            <SearchIconWrapper>
              <SearchIcon />
            </SearchIconWrapper>
            <StyledInputBase
              placeholder="URL"
              inputProps={{ "aria-label": "url" }}
              onChange={handleTargetInput}
              value={target}
              onKeyPress={(e) => {
                if (e.key === "Enter") {
                  window.navigate.navigate({ action: "goto", target: target });
                }
              }}
            />
          </Search>
          <IconButton
            size="middle"
            edge="start"
            color="inherit"
            sx={{ mr: 0 }}
            onClick={() => {
              window.navigate.navigate({ action: "goto", target: target });
            }}
          >
            <Surfing />
          </IconButton>
          <TextField
            size="middle"
            edge="start"
            // color="inherit"
            // sx={{ mr: 2 }}
            value={stdout}
            fullWidth
            label="ape-node"
            disabled
            color="secondary"
            // id="fullWidth"
          />
        </Toolbar>
      </AppBar>
    </Box>
  );
}
