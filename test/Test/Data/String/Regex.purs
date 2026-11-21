module Test.Data.String.Regex (testStringRegex) where

import Data.String.Regex

import Data.Array.NonEmpty (NonEmptyArray, fromArray)
import Data.Either (isLeft)
import Data.Maybe (Maybe(..), fromJust)
import Data.String.Regex.Flags (dotAll, global, ignoreCase, noFlags)
import Data.String.Regex.Unsafe (unsafeRegex)
import Effect (Effect)
import Effect.Console (log)
import Partial.Unsafe (unsafePartial)
import Prelude (type (~>), Unit, discard, not, show, ($), (<<<), (<>), (==))
import Test.Assert (assert)

testStringRegex :: Effect Unit
testStringRegex = do
  log "regex"
  assert $ test (unsafeRegex "^a" noFlags) "abc"
  assert $ not (test (unsafeRegex "^b" noFlags) "abc")
  assert $ isLeft (regex "+" noFlags)

  log "flags"
  assert $ "quxbarfoobaz" == replace (unsafeRegex "foo" noFlags) "qux" "foobarfoobaz"
  assert $ "quxbarquxbaz" == replace (unsafeRegex "foo" global) "qux" "foobarfoobaz"
  assert $ "quxbarquxbaz" == replace (unsafeRegex "foo" (global <> ignoreCase)) "qux" "foobarFOObaz"
  assert $ "quxbarfoobaz" == replace (unsafeRegex ".foo" dotAll) "qux" "\nfoobarfoobaz"

  log "match"
  assert $ match (unsafeRegex "^abc$" noFlags) "abc" == Just (nea [Just "abc"])
  assert $ match (unsafeRegex "^abc$" noFlags) "xyz" == Nothing

  log "replace"
  assert $ replace (unsafeRegex "-" noFlags) "!" "a-b-c" == "a!b-c"

  log "replace'"
  assert $ replace' (unsafeRegex "-" noFlags) (\_ _ -> "!") "a-b-c" == "a!b-c"
  assert $ replace' (unsafeRegex "(foo)(bar)?" noFlags) (\_ xs -> show xs) "<>" == "<>"
  assert $ replace' (unsafeRegex "(foo)(bar)?" noFlags) (\_ xs -> show xs) "<foo>" == "<[(Just \"foo\"),Nothing]>"
  assert $ replace' (unsafeRegex "(foo)(bar)?" noFlags) (\_ xs -> show xs) "<foobar>" == "<[(Just \"foo\"),(Just \"bar\")]>"
  assert $ replace' (unsafeRegex "@(?<username>\\w+)" noFlags) (\_ xs -> show xs) "@purescript" == "[(Just \"purescript\")]"

  log "search"
  assert $ search (unsafeRegex "b" noFlags) "abc" == Just 1
  assert $ search (unsafeRegex "d" noFlags) "abc" == Nothing

  log "split"
  assert $ split (unsafeRegex "" noFlags) "" == []
  assert $ split (unsafeRegex "" noFlags) "abc" == ["a", "b", "c"]
  assert $ split (unsafeRegex "b" noFlags) "" == [""]
  assert $ split (unsafeRegex "b" noFlags) "abc" == ["a", "c"]

  log "test"
  -- Ensure that we have referential transparency for calls to 'test'. No
  -- global state should be maintained between these two calls:
  let pattern = unsafeRegex "a" (parseFlags "g")
  assert $ test pattern "a"
  assert $ test pattern "a"


  log "lookaround and backreferences"
  assert $ test (unsafeRegex "a(?=b)" noFlags) "ab"
  assert $ not (test (unsafeRegex "a(?=b)" noFlags) "ac")
  assert $ test (unsafeRegex "(?<=a)b" noFlags) "ab"
  assert $ test (unsafeRegex "(?<!a)b" noFlags) "cb"
  assert $ test (unsafeRegex "(a)\\1" noFlags) "aa"
  assert $ match (unsafeRegex "(?<first>x)(y)\\k<first>" noFlags) "xyx" == Just (nea [Just "xyx", Just "x", Just "y"])
  assert $ match (unsafeRegex "(a*)()(b)?" noFlags) "aa" == Just (nea [Just "aa", Just "aa", Just "", Nothing])

  log "Unicode indices and flags"
  assert $ search (unsafeRegex "b" (parseFlags "u")) "😀b" == Just 2
  assert $ match (unsafeRegex "." (parseFlags "gu")) "😀x" == Just (nea [Just "😀", Just "x"])
  assert $ not (test (unsafeRegex "." noFlags) "\x2028")
  assert $ test (unsafeRegex "^a$" (parseFlags "m")) "x\ra\ry"
  assert $ not (test (unsafeRegex "a" (parseFlags "y")) "ba")
  assert $ flags (unsafeRegex "a" (parseFlags "y")) == parseFlags "y"
  assert $ flags (unsafeRegex "a" noFlags) == noFlags

  log "replacement and split contracts"
  assert $ replace (unsafeRegex "(a)(b)?" noFlags) "$1/$2/$$/$&" "a" == "a//$/a"
  assert $ replace (unsafeRegex "(?<first>x)(y)" noFlags) "$<first>-$2" "xy" == "x-y"
  assert $ split (unsafeRegex "(,)" noFlags) "a,b" == ["a", ",", "b"]

nea :: Array ~> NonEmptyArray
nea = unsafePartial fromJust <<< fromArray
