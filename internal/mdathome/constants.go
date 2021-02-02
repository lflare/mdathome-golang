package mdathome

////
// Client settings
///

// ClientVersion is the description of the version of the working branch when compiled
var ClientVersion string

// ClientSpecification is the integer version of the official specification this client is supposed to work against
const ClientSpecification int = 23

// Backend settings (Swap to use testnet)
const apiBackend string = "https://api.mangadex.network" // "https://mangadex-test.net"
