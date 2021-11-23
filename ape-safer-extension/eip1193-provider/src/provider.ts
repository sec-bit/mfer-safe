import JsonRpcProvider from "@json-rpc-tools/provider";
import { formatJsonRpcRequest } from "@json-rpc-tools/utils";

import { IEthereumProvider, ProviderAccounts, RequestArguments } from "./types";

export class EthereumProvider extends JsonRpcProvider
  implements IEthereumProvider {
  public enable(): Promise<ProviderAccounts> {
    console.log("enable")
    return this.request(formatJsonRpcRequest("eth_accounts", []));
  }

  public send(args): Promise<ProviderAccounts> {
    return this.request(args);
  }
  
  public isConnected() {
    console.log("is connected")
    return true;
  }

  _metamask = {
    "isUnlocked" : function(): boolean {
      return true;
    }
  }

  isMetaMask = true;
  chainId = this.request(formatJsonRpcRequest("eth_chainId", []));

  // public send(request: any, callback: (error: any, response?: any) => void): void {
  //   if (!request) callback('Undefined request');
  //   this.request(request)
  //     .then((result) => callback(null, { jsonrpc: '2.0', id: request.id, result }))
  //     .catch((error) => callback(error, null));
  // }
}
