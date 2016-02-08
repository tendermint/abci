{-# LANGUAGE BangPatterns, DeriveDataTypeable, FlexibleInstances, MultiParamTypeClasses #-}
{-# OPTIONS_GHC  -fno-warn-unused-imports #-}
module Types.CodeType (CodeType(..)) where
import Prelude ((+), (/), (.))
import qualified Prelude as Prelude'
import qualified Data.Typeable as Prelude'
import qualified Data.Data as Prelude'
import qualified Text.ProtocolBuffers.Header as P'

data CodeType = OK
              | InternalError
              | Unauthorized
              | InsufficientFees
              | UnknownRequest
              | EncodingError
              | BadNonce
              | UnknownAccount
              | InsufficientFunds
              deriving (Prelude'.Read, Prelude'.Show, Prelude'.Eq, Prelude'.Ord, Prelude'.Typeable, Prelude'.Data)

instance P'.Mergeable CodeType

instance Prelude'.Bounded CodeType where
  minBound = OK
  maxBound = InsufficientFunds

instance P'.Default CodeType where
  defaultValue = OK

toMaybe'Enum :: Prelude'.Int -> P'.Maybe CodeType
toMaybe'Enum 0 = Prelude'.Just OK
toMaybe'Enum 1 = Prelude'.Just InternalError
toMaybe'Enum 2 = Prelude'.Just Unauthorized
toMaybe'Enum 3 = Prelude'.Just InsufficientFees
toMaybe'Enum 4 = Prelude'.Just UnknownRequest
toMaybe'Enum 5 = Prelude'.Just EncodingError
toMaybe'Enum 6 = Prelude'.Just BadNonce
toMaybe'Enum 7 = Prelude'.Just UnknownAccount
toMaybe'Enum 8 = Prelude'.Just InsufficientFunds
toMaybe'Enum _ = Prelude'.Nothing

instance Prelude'.Enum CodeType where
  fromEnum OK = 0
  fromEnum InternalError = 1
  fromEnum Unauthorized = 2
  fromEnum InsufficientFees = 3
  fromEnum UnknownRequest = 4
  fromEnum EncodingError = 5
  fromEnum BadNonce = 6
  fromEnum UnknownAccount = 7
  fromEnum InsufficientFunds = 8
  toEnum = P'.fromMaybe (Prelude'.error "hprotoc generated code: toEnum failure for type Types.CodeType") . toMaybe'Enum
  succ OK = InternalError
  succ InternalError = Unauthorized
  succ Unauthorized = InsufficientFees
  succ InsufficientFees = UnknownRequest
  succ UnknownRequest = EncodingError
  succ EncodingError = BadNonce
  succ BadNonce = UnknownAccount
  succ UnknownAccount = InsufficientFunds
  succ _ = Prelude'.error "hprotoc generated code: succ failure for type Types.CodeType"
  pred InternalError = OK
  pred Unauthorized = InternalError
  pred InsufficientFees = Unauthorized
  pred UnknownRequest = InsufficientFees
  pred EncodingError = UnknownRequest
  pred BadNonce = EncodingError
  pred UnknownAccount = BadNonce
  pred InsufficientFunds = UnknownAccount
  pred _ = Prelude'.error "hprotoc generated code: pred failure for type Types.CodeType"

instance P'.Wire CodeType where
  wireSize ft' enum = P'.wireSize ft' (Prelude'.fromEnum enum)
  wirePut ft' enum = P'.wirePut ft' (Prelude'.fromEnum enum)
  wireGet 14 = P'.wireGetEnum toMaybe'Enum
  wireGet ft' = P'.wireGetErr ft'
  wireGetPacked 14 = P'.wireGetPackedEnum toMaybe'Enum
  wireGetPacked ft' = P'.wireGetErr ft'

instance P'.GPB CodeType

instance P'.MessageAPI msg' (msg' -> CodeType) CodeType where
  getVal m' f' = f' m'

instance P'.ReflectEnum CodeType where
  reflectEnum
   = [(0, "OK", OK), (1, "InternalError", InternalError), (2, "Unauthorized", Unauthorized),
      (3, "InsufficientFees", InsufficientFees), (4, "UnknownRequest", UnknownRequest), (5, "EncodingError", EncodingError),
      (6, "BadNonce", BadNonce), (7, "UnknownAccount", UnknownAccount), (8, "InsufficientFunds", InsufficientFunds)]
  reflectEnumInfo _
   = P'.EnumInfo (P'.makePNF (P'.pack ".types.CodeType") [] ["Types"] "CodeType") ["Types", "CodeType.hs"]
      [(0, "OK"), (1, "InternalError"), (2, "Unauthorized"), (3, "InsufficientFees"), (4, "UnknownRequest"), (5, "EncodingError"),
       (6, "BadNonce"), (7, "UnknownAccount"), (8, "InsufficientFunds")]

instance P'.TextType CodeType where
  tellT = P'.tellShow
  getT = P'.getRead