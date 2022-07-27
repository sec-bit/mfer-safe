#![cfg_attr(
    all(not(debug_assertions), target_os = "windows"),
    windows_subsystem = "windows"
)]

use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;
use std::sync::Mutex;
use tauri::api::path::resolve_path;
use tauri::api::path::BaseDirectory;
use tauri::api::process::Command;
use tauri::api::process::CommandChild;
use tauri::api::process::CommandEvent;
use tauri::Env;
use tauri::Manager;
use tokio::sync::mpsc;
use tokio::sync::mpsc::Sender;

#[derive(Debug)]
struct SpawnApeNode {
    args: Mutex<ApeNodeArgs>,
    child: Mutex<Option<CommandChild>>,
    tx: Sender<CommandEvent>,
    config_path: PathBuf,
}

impl SpawnApeNode {
    fn new(
        binary_path: String,
        args: ApeNodeArgs,
        tx: Sender<CommandEvent>,
        config_path: PathBuf,
    ) -> SpawnApeNode {
        let (mut rx, child) = Command::new_sidecar(&binary_path)
            .expect("failed to load sidecar binary")
            .args(args.to_args())
            .spawn()
            .expect("failed to execute process");
        let tx2 = tx.clone();
        tauri::async_runtime::spawn(async move {
            while let Some(event) = rx.recv().await {
                tx2.send(event).await.unwrap();
            }
        });
        SpawnApeNode {
            child: Mutex::new(Some(child)),
            args: Mutex::new(args),
            tx,
            config_path,
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
struct ApeNodeArgs {
    impersonated_account: String,
    web3_rpc: String,
    listen_host_port: String,
    key_cache_file_path: String,
    log_file_path: String,
    batch_size: i64,
}

impl ApeNodeArgs {
    fn new(config_path: PathBuf) -> ApeNodeArgs {
        let args = ApeNodeArgs::default();
        args.load_config(config_path)
    }

    fn to_args(&self) -> Vec<String> {
        let mut args = Vec::new();
        args.push("-account".to_string());
        args.push(self.impersonated_account.clone());
        args.push("-logpath".to_string());
        args.push(self.log_file_path.clone());
        args.push("-upstream".to_string());
        args.push(self.web3_rpc.to_string());
        args.push("-listen".to_string());
        args.push(self.listen_host_port.to_string());
        if !self.key_cache_file_path.is_empty() {
            args.push("-keycache".to_string());
            args.push(self.key_cache_file_path.to_string());
        }
        args.push("-batchsize".to_string());
        args.push(self.batch_size.to_string());
        args
    }

    fn load_config(&self, config_path: PathBuf) -> ApeNodeArgs {
        let default_config = ApeNodeArgs {
            impersonated_account: "0x0000000000000000000000000000000000000000".to_string(),
            web3_rpc: "wss://mainnet.infura.io/ws/v3/547e677c41724f8fa8c4bfde1aecef20".to_string(),
            listen_host_port: "127.0.0.1:10545".to_string(),
            key_cache_file_path: "".to_string(),
            log_file_path: "".to_string(),
            batch_size: 100,
        };

        // let config_path = self.get_config_path(context);
        let config_str = std::fs::read_to_string(&config_path).unwrap_or_default();
        let decoded: ApeNodeArgs =
            serde_json::from_str(config_str.as_str()).unwrap_or(default_config);
        decoded
    }

    fn save_config(&self, config_path: PathBuf) {
        if let Some(p) = config_path.parent() {
            match fs::create_dir_all(p) {
                Ok(_) => {}
                Err(e) => {
                    println!("failed to create config directory: {}", e);
                }
            }
        };
        let config = serde_json::to_string_pretty(&self).unwrap();
        match fs::write(config_path, config) {
            Ok(_) => (),
            Err(e) => {
                println!("failed to write config: {}", e);
            }
        }
    }
}

const BIN_PATH: &'static str = "mfer-node";

#[tauri::command]
fn restart_mfer_node(
    mfer_node_args: ApeNodeArgs,
    state: tauri::State<'_, SpawnApeNode>,
    app: tauri::AppHandle,
) -> bool {
    // println!("{:#?}, {:#?}", mfer_node_args, state);
    let child = state.child.lock().unwrap().take();
    child.unwrap().kill().unwrap();

    let args = mfer_node_args.to_args();
    let (mut rx, child) = Command::new_sidecar(BIN_PATH.to_string())
        .expect("failed to load sidecar binary")
        .args(args.clone())
        .spawn()
        .expect("failed to execute process");
    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            let app_state = app.state::<SpawnApeNode>();
            app_state.tx.clone().send(event).await.unwrap();
        }
    });
    mfer_node_args.save_config(state.config_path.clone());
    *state.args.lock().unwrap() = mfer_node_args;
    *state.child.lock().unwrap() = Some(child);
    true
}

#[tauri::command]
fn get_mfer_node_args(state: tauri::State<'_, SpawnApeNode>) -> ApeNodeArgs {
    state.args.lock().unwrap().clone()
}

fn main() {
    let (tx, mut rx) = mpsc::channel(1000);

    let context = tauri::generate_context!();
    let config_path = resolve_path(
        context.config(),
        context.package_info(),
        &Env::default(),
        ".config/mfersafe.json",
        Some(BaseDirectory::Home),
    )
    .expect("resolve path failed");
    println!("config path: {:?}", config_path);

    let mfer_node_args = ApeNodeArgs::new(config_path.clone());
    let mfer_node = SpawnApeNode::new(
        BIN_PATH.to_string(),
        mfer_node_args,
        tx.clone(),
        config_path.clone(), // just for saving config path
    );

    tauri::Builder::default()
        .setup(|app| {
            let main_window = app.get_window("main").unwrap();
            tauri::async_runtime::spawn(async move {
                while let Some(event) = rx.recv().await {
                    match event {
                        CommandEvent::Stdout(line) => {
                            main_window
                                .emit("mfernode-event", Some(format!("{:?}", line)))
                                .expect("failed to emit event");
                        }
                        CommandEvent::Stderr(line) => {
                            main_window
                                .emit("mfernode-event", Some(format!("StdErr: {:?}", line)))
                                .expect("failed to emit event");
                        }
                        unhandeled_line => {
                            println!("unhandeled line: {:#?}", unhandeled_line);
                        }
                    }
                }
            });
            Ok(())
        })
        .menu(if cfg!(target_os = "macos") {
            tauri::Menu::os_default(&context.package_info().name)
        } else {
            tauri::Menu::default()
        })
        .manage(mfer_node)
        .invoke_handler(tauri::generate_handler![
            restart_mfer_node,
            get_mfer_node_args,
        ])
        .run(context)
        .expect("error while running tauri application");
}
