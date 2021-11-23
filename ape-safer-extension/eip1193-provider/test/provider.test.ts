import EthereumProvider from "../src";

describe("EthereumProvider", () => {
  it("eth_chainId", async () => {
    const provider = new EthereumProvider(`https://rpc.slock.it/mainnet`);
    const result = await provider.request({ method: "eth_chainId" });
    expect(!!result).toBeTruthy();
    expect(result).toEqual("0x1");
  });
});
