// Modules to control application life and create native browser window
const {
  app,
  BrowserView,
  BrowserWindow,
  ipcMain,
  session,
} = require("electron");
const fetch = require("node-fetch");
const Store = require("electron-store");

const path = require("path");
const needRedir = {};
const needExclude = {};
const filter = {
  urls: ["https://*/*", "http://*/*"],
};

const apeNodePath = path.join(app.getAppPath(), "bin", "ape-safer");
const spawn = require("child_process").spawn;

const store = new Store();

var upstream_rpc = store.get("settings.upstream_rpc", "ws://localhost:8546");
var apesafer_server = store.get(
  "settings.apesafer_server",
  "http://127.0.0.1:10545"
);
var impersonated_account = store.get(
  "settings.impersonated_account",
  "0x0000000000000000000000000000000000000000"
);
var listen = store.get("settings.listen", "127.0.0.1:10545");

function createWindow() {
  const mainWindow = new BrowserWindow({
    width: 1000,
    height: 1000,
  });
  const subWindow = new BrowserWindow({
    webPreferences: {
      preload: path.join(app.getAppPath(), "navigationbar-preload.js"),
    },
  });

  mainWindow.addTabbedWindow(subWindow);
  subWindow.loadFile(path.join(__dirname, "frontend", "index.html"));
  mainWindow.show();
  // subWindow.webContents.openDevTools();

  return mainWindow;
}

function createView(mainWindow) {
  const navigationView = new BrowserView({
    webPreferences: {
      preload: path.join(app.getAppPath(), "navigationbar-preload.js"),
    },
  });

  const dappView = new BrowserView({
    webPreferences: {
      // nodeIntegration: true,
      preload: path.join(app.getAppPath(), "preload.js"),
    },
  });
  mainWindow.addBrowserView(navigationView);
  mainWindow.addBrowserView(dappView);

  const navigationBarWidth = 64;
  var resize = () => {
    navigationView.setBounds({
      x: 0,
      y: 0,
      width: mainWindow.getBounds().width,
      height: navigationBarWidth,
    });
    dappView.setBounds({
      x: 0,
      y: navigationBarWidth,
      width: mainWindow.getBounds().width,
      height: mainWindow.getBounds().height - navigationBarWidth - 55,
    });
  };
  mainWindow.on("resize", resize);
  mainWindow.once("ready-to-show", resize);

  navigationView.webContents.loadFile(
    path.join(__dirname, "frontend", "index.html")
  );
  navigationView.webContents.executeJavaScript(
    "window.location.assign(window.location.href+'?page=navigationbar');0"
  );

  // dappView.webContents.openDevTools();
  // navigationView.webContents.openDevTools();
  dappView.webContents.setWindowOpenHandler((details) => {
    dappView.webContents.loadURL(details.url);
    console.log("loadURL:", details.url);
    return { action: "deny" };
  });

  dappView.webContents.on("dom-ready", () => {
    mainWindow.setTitle(dappView.webContents.getTitle());
    var currentURL = dappView.webContents.getURL();
    console.log("current url:", currentURL);
    navigationView.webContents.send("target", currentURL);
  });
  return { dappView, navigationView };
}

function handleNavigationAction(dappView) {
  ipcMain.handle("navigation", (event, args) => {
    console.log(args);
    switch (args.action) {
      case "goto":
        var targetURL = args.target;
        var pattern = /^((http|https):\/\/)/;
        if (!pattern.test(targetURL)) {
          targetURL = "https://" + targetURL;
        }
        dappView.webContents.loadURL(targetURL);
        break;
      case "forward":
        dappView.webContents.goForward();
        break;
      case "backward":
        dappView.webContents.goBack();
        break;
      case "refresh":
        dappView.webContents.reload();
        break;
      case "clearStorage":
        session.defaultSession.clearStorageData();
        break;
    }
  });
}

function spawnApeNode(account, rpc, listen, navigationView) {
  navigationView.webContents.send("stdout", "Reving up the node...");
  var child = spawn(apeNodePath, [
    "-account",
    account,
    "-upstream",
    rpc,
    "-listen",
    listen,
  ]);
  child.stdout.on("data", (data) => {
    var out = `${data}`;
    if (out.includes("Downstream")) {
      return;
    }
    console.log("stdout:", out);
    if (out.indexOf("[Update]") >= 0) {
      navigationView.webContents.send("stdout", `${data}`);
    }
  });
  child.stderr.on("data", (data) => {
    var out = `${data}`;
    if (out == "\n" || out == "\r") {
      return;
    }
    console.log(`stderr: ${data}`);
    if (out.indexOf("batch req") >= 0) {
      navigationView.webContents.send("stdout", `${data}`);
    }
  });
  return child;
}

function prepareNetwork(ape_node_rpc) {
  session.defaultSession.webRequest.onHeadersReceived(
    filter,
    (details, callback) => {
      if (details.responseHeaders["content-security-policy"] != undefined) {
        console.log("csp:", details.responseHeaders["content-security-policy"]);
        details.responseHeaders["content-security-policy"] = "";
        return callback({ responseHeaders: details.responseHeaders });
      }
      callback({});
    }
  );

  session.defaultSession.webRequest.onBeforeRequest(
    filter,
    (details, callback) => {
      // console.log("on before request", details.url);
      if (
        details.uploadData == undefined ||
        details.uploadData.length == 0 ||
        details.uploadData[0].bytes == undefined
      ) {
        return callback({});
      }
      if (needRedir[details.url]) {
        console.log("redir:", details.url, "to:", ape_node_rpc);
        // var ret = details.uploadData[0].bytes.toString("utf8");
        // console.log(ret);
        return callback({ redirectURL: ape_node_rpc });
      }

      if (needExclude[details.url] || details.url.indexOf(ape_node_rpc) >= 0) {
        // console.log("excluded:", details.url);
        needExclude[details.url] = true;
        return callback({});
      }

      if (details.url !== ape_node_rpc) {
        try {
          // console.log(details.uploadData[0].bytes);
          var ret = details.uploadData[0].bytes.toString("utf8");
          // console.log(ret);
          var method = JSON.parse(ret).method
            ? JSON.parse(ret).method
            : JSON.parse(ret)[0].method;

          if (method.indexOf("eth_") == 0) {
            console.log("redir: ", details.url, " to: ", ape_node_rpc);
            needRedir[details.url] = true;
            return callback({ redirectURL: ape_node_rpc });
          }
        } catch (e) {
          needExclude[details.url] = true;
          console.log("json parse error", details.url, "excluded");
          return callback({});
        }
      }
    }
  );
  session.defaultSession.setProxy({ mode: "system" });
}

app.whenReady().then(() => {
  var mainWindow = createWindow();
  var { dappView, navigationView } = createView(mainWindow);
  handleNavigationAction(dappView);
  var child = spawnApeNode(
    impersonated_account,
    upstream_rpc,
    listen,
    navigationView
  );

  ipcMain.on("settings", (event, args) => {
    if (args.setweb3rpc != undefined && args.setweb3rpc != "") {
      child.kill();
      child = spawnApeNode(
        args.impersonatedAccount,
        args.setweb3rpc,
        args.setlistenhostport,
        navigationView
      );
      store.set({
        settings: {
          upstream_rpc: args.setweb3rpc,
          apesafer_server: "http://" + args.setlistenhostport,
          impersonated_account: args.impersonatedAccount,
          listen: args.setlistenhostport,
        },
      });
    }
  });
  ipcMain.handle("settings", (event, args) => {
    return store.get("settings");
  });
  ipcMain.handle("eth:fetch", (event, args) => {
    var ret = fetch(apesafer_server, args);
    var result = ret.then((res) => res.json()).then((data) => data.result);
    return result;
  });

  prepareNetwork(apesafer_server);

  app.on("activate", function () {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (BrowserWindow.getAllWindows().length === 0) {
      var mainWindow = createWindow();
      mainWindow.addBrowserView(navigationView);
      mainWindow.addBrowserView(dappView);
    }
  });
});

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on("window-all-closed", function () {
  if (process.platform !== "darwin") app.quit();
});

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and require them here.
