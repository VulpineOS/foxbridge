package bridge

import (
	"fmt"
	"log"
)

const (
	defaultDeterministicSeed   uint32 = 1
	defaultDeterministicTimeMS int64  = 1700000000000
)

type DeterministicConfig struct {
	TimeMS int64
	Seed   uint32
}

func (b *Bridge) SetDeterministicConfig(cfg DeterministicConfig) {
	script := buildDeterministicScript(cfg)
	b.deterministicMu.Lock()
	b.deterministicScript = script
	clear(b.deterministicApplied)
	b.deterministicMu.Unlock()
}

func (b *Bridge) applyDeterministicPrelude(cdpSessionID string) {
	if cdpSessionID == "" {
		return
	}

	b.deterministicMu.Lock()
	script := b.deterministicScript
	if script == "" || b.deterministicApplied[cdpSessionID] {
		b.deterministicMu.Unlock()
		return
	}
	b.deterministicApplied[cdpSessionID] = true
	b.deterministicMu.Unlock()

	_, err := b.callJuggler(cdpSessionID, "Page.addScriptToEvaluateOnNewDocument", map[string]interface{}{
		"script": script,
	})
	if err != nil {
		log.Printf("[deterministic] addScriptToEvaluateOnNewDocument failed for session %s: %v", cdpSessionID, err)
	}

	_, err = b.callJuggler(cdpSessionID, "Runtime.evaluate", map[string]interface{}{
		"expression":    script,
		"returnByValue": true,
	})
	if err != nil {
		log.Printf("[deterministic] Runtime.evaluate bootstrap failed for session %s: %v", cdpSessionID, err)
	}
}

func buildDeterministicScript(cfg DeterministicConfig) string {
	if cfg.TimeMS == 0 && cfg.Seed == 0 {
		return ""
	}

	timeMS := cfg.TimeMS
	if timeMS == 0 {
		timeMS = defaultDeterministicTimeMS
	}
	seed := cfg.Seed
	if seed == 0 {
		seed = defaultDeterministicSeed
	}

	return fmt.Sprintf(`(() => {
  if (globalThis.__foxbridgeDeterministicApplied) return;
  Object.defineProperty(globalThis, "__foxbridgeDeterministicApplied", {
    value: true,
    configurable: false,
    enumerable: false,
    writable: false
  });

  let __foxbridgeTick = 0;
  let __foxbridgeSeed = %d >>> 0;
  const __foxbridgeBaseTime = %d;
  const __foxbridgeNextTick = () => {
    const value = __foxbridgeTick;
    __foxbridgeTick += 1;
    return value;
  };
  const __foxbridgeNextUint32 = () => {
    __foxbridgeSeed = ((__foxbridgeSeed * 1664525) + 1013904223) >>> 0;
    return __foxbridgeSeed;
  };
  const __foxbridgeCrypto = globalThis.crypto;
  const __foxbridgeNativeGetRandomValues = __foxbridgeCrypto && typeof __foxbridgeCrypto.getRandomValues === "function"
    ? __foxbridgeCrypto.getRandomValues.bind(__foxbridgeCrypto)
    : null;

  Object.defineProperty(Date, "now", {
    configurable: true,
    value: () => __foxbridgeBaseTime + __foxbridgeNextTick()
  });

  if (globalThis.performance && typeof globalThis.performance.now === "function") {
    Object.defineProperty(globalThis.performance, "now", {
      configurable: true,
      value: () => __foxbridgeNextTick()
    });
  }

  Object.defineProperty(Math, "random", {
    configurable: true,
    value: () => __foxbridgeNextUint32() / 4294967296
  });

  if (__foxbridgeCrypto && __foxbridgeNativeGetRandomValues) {
    Object.defineProperty(__foxbridgeCrypto, "getRandomValues", {
      configurable: true,
      value: (typedArray) => {
        if (!typedArray || typeof typedArray.length !== "number") {
          return __foxbridgeNativeGetRandomValues(typedArray);
        }
        for (let i = 0; i < typedArray.length; i++) {
          const next = __foxbridgeNextUint32();
          if (typeof typedArray[i] === "bigint") {
            const upper = BigInt(next);
            const lower = BigInt(__foxbridgeNextUint32());
            const bits = BigInt((typedArray.BYTES_PER_ELEMENT || 8) * 8);
            typedArray[i] = ((upper << 32n) | lower) & ((1n << bits) - 1n);
            continue;
          }
          switch (typedArray.BYTES_PER_ELEMENT) {
            case 1:
              typedArray[i] = next & 0xff;
              break;
            case 2:
              typedArray[i] = next & 0xffff;
              break;
            default:
              typedArray[i] = next >>> 0;
              break;
          }
        }
        return typedArray;
      }
    });
  }
})();`, seed, timeMS)
}
