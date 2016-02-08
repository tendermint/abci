import Network.Socket
import Data.Maybe
import Data.Char
import Data.Binary.Get
import Text.ProtocolBuffers(messageGet,messagePut,Utf8(..),defaultValue)
import Text.ProtocolBuffers.Header as P'
import qualified Data.ByteString.Lazy.UTF8 as U(fromString)
import qualified Data.ByteString.Lazy.Char8 as C(pack,unpack,length,cons)
import Types
import qualified Types.Request as TReq
import qualified Types.Response as TRes
import qualified Types.MessageType as TMsg

main :: IO()
main = do
	sock <- socket AF_INET Stream 0
	setSocketOption sock ReuseAddr 1
	bind sock (SockAddrInet 46658 iNADDR_ANY)
	listen sock 2
	mainLoop sock

mainLoop :: Socket -> IO()
mainLoop sock = do
	putStrLn "waiting for conn ..."
	conn <- accept sock
	runConn conn
	return () --mainLoop sock

runConn :: (Socket, SockAddr) -> IO()
runConn (sock, _) = do
	-- read off the socket
	-- throw away the first two bytes for now (the length prefix)
	reqString <- readMessage sock
	print reqString

	-- string -> bytestring
	let reqBytes = C.pack reqString

	-- unmarshal the protobuf msg request
  	req <- case messageGet reqBytes :: Either String (TReq.Request, P'.ByteString) of
               Left m -> error ("Failed to parse msg .\n"++m)
               Right (m,_) -> return m
	print req

	-- build a response
 	let resp = TRes.Response { 
		TRes.type' = TReq.type' req, 
		TRes.data' = TReq.data' req, 
		TRes.code = Nothing, 
		TRes.error = Nothing, 
		TRes.log = Nothing
	}

	-- marshal response
 	let respBytes = messagePut resp
 	print respBytes

 	-- prefix with length
 	let respBytes1 = C.cons (chr (fromInt64ToInt $ C.length respBytes)) respBytes
 	print respBytes1
 
 	-- prefix with length of length
 	let respBytes2 = C.cons (chr 1) respBytes1
 	print respBytes2
 
 	-- bytestring -> string
 	let respString = C.unpack respBytes2
 	print respString
 
 	-- send response on socket
 	send sock respString

	-- build a response
 	let resp = TRes.Response { 
		TRes.type' = TMsg.Flush,
		TRes.data' = Nothing,
		TRes.code = Nothing, 
		TRes.error = Nothing, 
		TRes.log = Nothing
	}

	-- marshal response
 	let respBytes = messagePut resp
 	print respBytes

 	-- prefix with length
 	let respBytes1 = C.cons (chr (fromInt64ToInt $ C.length respBytes)) respBytes
 	print respBytes1
 
 	-- prefix with length of length
 	let respBytes2 = C.cons (chr 1) respBytes1
 	print respBytes2
 
 	-- bytestring -> string
 	let respString = C.unpack respBytes2
 	print respString
 
 	-- send response on socket
 	send sock respString

	return ()


readMessage :: Socket -> IO String
readMessage sock = do
	lengthOfLength <- recv sock 1
	print lengthOfLength
	print (bigEndianString8ToInt lengthOfLength)
	lengthString <- readMessageOfLength sock (bigEndianString8ToInt lengthOfLength)
	print lengthString
	print (padStringBE lengthString 4)
	print (bigEndianString32ToInt (padStringBE lengthString 4))
	readMessageOfLength sock (bigEndianString32ToInt (padStringBE lengthString 4))

-- TODO: be a man (we might not get all of length in one recv)
readMessageOfLength :: Socket -> Int -> IO String
readMessageOfLength sock length = recv sock length



bigEndianString32ToInt :: String -> Int
bigEndianString32ToInt s = fromIntegral (runGet getWord32be (C.pack s)) :: Int

bigEndianString8ToInt :: String -> Int
bigEndianString8ToInt s = fromIntegral (runGet getWord8 (C.pack s)) :: Int

fromInt64ToInt :: Int64 -> Int
fromInt64ToInt = fromIntegral

-- pad a string with the 0 byte
padStringBE :: String -> Int -> String
padStringBE s n 
    | length s < n  = (replicate (n - length s) '\NUL') ++ s 
    | otherwise     = s
