{-# LANGUAGE BangPatterns, DeriveDataTypeable, FlexibleInstances, MultiParamTypeClasses #-}
{-# OPTIONS_GHC  -fno-warn-unused-imports #-}
module Types.MessageType (MessageType(..)) where
import Prelude ((+), (/), (.))
import qualified Prelude as Prelude'
import qualified Data.Typeable as Prelude'
import qualified Data.Data as Prelude'
import qualified Text.ProtocolBuffers.Header as P'

data MessageType = NullMessage
                 | Echo
                 | Flush
                 | Info
                 | SetOption
                 | Exception
                 | AppendTx
                 | CheckTx
                 | GetHash
                 | Query
                 deriving (Prelude'.Read, Prelude'.Show, Prelude'.Eq, Prelude'.Ord, Prelude'.Typeable, Prelude'.Data)

instance P'.Mergeable MessageType

instance Prelude'.Bounded MessageType where
  minBound = NullMessage
  maxBound = Query

instance P'.Default MessageType where
  defaultValue = NullMessage

toMaybe'Enum :: Prelude'.Int -> P'.Maybe MessageType
toMaybe'Enum 0 = Prelude'.Just NullMessage
toMaybe'Enum 1 = Prelude'.Just Echo
toMaybe'Enum 2 = Prelude'.Just Flush
toMaybe'Enum 3 = Prelude'.Just Info
toMaybe'Enum 4 = Prelude'.Just SetOption
toMaybe'Enum 5 = Prelude'.Just Exception
toMaybe'Enum 17 = Prelude'.Just AppendTx
toMaybe'Enum 18 = Prelude'.Just CheckTx
toMaybe'Enum 19 = Prelude'.Just GetHash
toMaybe'Enum 20 = Prelude'.Just Query
toMaybe'Enum _ = Prelude'.Nothing

instance Prelude'.Enum MessageType where
  fromEnum NullMessage = 0
  fromEnum Echo = 1
  fromEnum Flush = 2
  fromEnum Info = 3
  fromEnum SetOption = 4
  fromEnum Exception = 5
  fromEnum AppendTx = 17
  fromEnum CheckTx = 18
  fromEnum GetHash = 19
  fromEnum Query = 20
  toEnum = P'.fromMaybe (Prelude'.error "hprotoc generated code: toEnum failure for type Types.MessageType") . toMaybe'Enum
  succ NullMessage = Echo
  succ Echo = Flush
  succ Flush = Info
  succ Info = SetOption
  succ SetOption = Exception
  succ Exception = AppendTx
  succ AppendTx = CheckTx
  succ CheckTx = GetHash
  succ GetHash = Query
  succ _ = Prelude'.error "hprotoc generated code: succ failure for type Types.MessageType"
  pred Echo = NullMessage
  pred Flush = Echo
  pred Info = Flush
  pred SetOption = Info
  pred Exception = SetOption
  pred AppendTx = Exception
  pred CheckTx = AppendTx
  pred GetHash = CheckTx
  pred Query = GetHash
  pred _ = Prelude'.error "hprotoc generated code: pred failure for type Types.MessageType"

instance P'.Wire MessageType where
  wireSize ft' enum = P'.wireSize ft' (Prelude'.fromEnum enum)
  wirePut ft' enum = P'.wirePut ft' (Prelude'.fromEnum enum)
  wireGet 14 = P'.wireGetEnum toMaybe'Enum
  wireGet ft' = P'.wireGetErr ft'
  wireGetPacked 14 = P'.wireGetPackedEnum toMaybe'Enum
  wireGetPacked ft' = P'.wireGetErr ft'

instance P'.GPB MessageType

instance P'.MessageAPI msg' (msg' -> MessageType) MessageType where
  getVal m' f' = f' m'

instance P'.ReflectEnum MessageType where
  reflectEnum
   = [(0, "NullMessage", NullMessage), (1, "Echo", Echo), (2, "Flush", Flush), (3, "Info", Info), (4, "SetOption", SetOption),
      (5, "Exception", Exception), (17, "AppendTx", AppendTx), (18, "CheckTx", CheckTx), (19, "GetHash", GetHash),
      (20, "Query", Query)]
  reflectEnumInfo _
   = P'.EnumInfo (P'.makePNF (P'.pack ".types.MessageType") [] ["Types"] "MessageType") ["Types", "MessageType.hs"]
      [(0, "NullMessage"), (1, "Echo"), (2, "Flush"), (3, "Info"), (4, "SetOption"), (5, "Exception"), (17, "AppendTx"),
       (18, "CheckTx"), (19, "GetHash"), (20, "Query")]

instance P'.TextType MessageType where
  tellT = P'.tellShow
  getT = P'.getRead