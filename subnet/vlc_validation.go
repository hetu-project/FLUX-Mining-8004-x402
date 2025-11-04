// Package subnet - VLC Protocol Validation
//
// This file implements the VLC (Vector Logical Clock) validation system for agent onboarding.
// Validators test whether agents correctly implement VLC causality by sending tasks that
// require additional information, verifying that VLC increments properly at each step.
package subnet

import (
	"fmt"
	"time"

	"github.com/hetu-project/FLUX-Mining-8004-x402/vlc"
)

// VLCValidationTest tracks the state and results of a VLC protocol validation
type VLCValidationTest struct {
	AgentID           string
	MinerAddress      string
	InitialClock      *vlc.Clock
	AfterFirstStep    *vlc.Clock
	AfterSecondStep   *vlc.Clock
	TestPassed        bool
	Score             uint8
	Timestamp         time.Time
	FailureReason     string
}

// VLCValidationResult contains the final validation outcome
type VLCValidationResult struct {
	AgentID       string
	ValidatorID   string
	Score         uint8
	Passed        bool
	Details       string
	Timestamp     time.Time
}

// ValidateAgentVLC performs a comprehensive VLC protocol test on a new agent.
// This test verifies that the agent correctly implements Vector Logical Clock causality
// by sending a task designed to trigger the NeedMoreInfo flow.
//
// Test Sequence:
//   1. Send ambiguous task that requires clarification
//   2. Verify agent responds with NeedMoreInfo and VLC increments by 1
//   3. Provide additional information
//   4. Verify agent provides final answer and VLC increments by 1 again
//   5. Validate VLC consistency throughout the process
//
// Returns VLCValidationTest with complete test results and score (0-100)
func (v *CoreValidator) ValidateAgentVLC(miner *CoreMiner, requestID string) *VLCValidationTest {
	test := &VLCValidationTest{
		AgentID:      miner.ID,
		MinerAddress: miner.ID,
		Timestamp:    time.Now(),
	}

	fmt.Printf("\n🔍 [%s] VLC Validation Test Starting\n", v.ID)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Agent: %s\n", miner.ID)
	fmt.Printf("Validator: %s\n", v.ID)
	fmt.Printf("Request ID: %s\n", requestID)
	fmt.Println()

	// STEP 1: Capture initial VLC state
	test.InitialClock = miner.GetCurrentClock().Copy()
	fmt.Printf("📊 Initial VLC State: %v\n", test.InitialClock)
	fmt.Println()

	// STEP 2: Send intentionally ambiguous task to trigger NeedMoreInfo
	ambiguousTask := "Calculate the optimal route"
	fmt.Printf("📤 [Step 1] Sending ambiguous task: \"%s\"\n", ambiguousTask)

	response1 := miner.ProcessInput(ambiguousTask, 1, requestID)

	// STEP 3: Verify first response requests more information
	fmt.Printf("📥 Agent Response: %s\n", response1.OutputType)
	if response1.OutputType != NeedMoreInfo {
		test.TestPassed = false
		test.Score = 0
		test.FailureReason = fmt.Sprintf("Expected NeedMoreInfo, got %s", response1.OutputType)
		fmt.Printf("❌ FAILED: %s\n", test.FailureReason)
		return test
	}
	fmt.Printf("✅ Correct: Agent requested additional information\n")
	fmt.Printf("   Info Request: \"%s\"\n", response1.InfoRequest)

	// STEP 4: Verify VLC incremented correctly for NeedMoreInfo (miner ID = 1)
	// Expected pattern: 0 → 2 (two increments: message enter + message leave)
	test.AfterFirstStep = response1.VLCClock.Copy()
	fmt.Printf("📊 VLC After Step 1 (NeedMoreInfo): %v\n", test.AfterFirstStep)

	if !v.verifyVLCIncrement(test.InitialClock, test.AfterFirstStep, 1, 2) {
		test.TestPassed = false
		test.Score = 40
		test.FailureReason = "VLC did not increment correctly on NeedMoreInfo response"
		fmt.Printf("❌ FAILED: %s\n", test.FailureReason)
		fmt.Printf("   Expected: increment by 2 on node 1 (message enter + message leave)\n")
		fmt.Printf("   Got: Initial[1]=%d, After[1]=%d\n",
			test.InitialClock.Values[1], test.AfterFirstStep.Values[1])
		return test
	}
	fmt.Printf("✅ Correct: VLC incremented properly [node 1: %d → %d] (message enter + message leave)\n",
		test.InitialClock.Values[1], test.AfterFirstStep.Values[1])
	fmt.Println()

	// STEP 5: Provide additional information
	additionalInfo := "Route from point A(0,0) to point B(10,10), avoid obstacles at (5,5)"
	fmt.Printf("📤 [Step 2] Providing additional info: \"%s\"\n", additionalInfo)

	response2 := miner.ProcessAdditionalInfo(ambiguousTask, additionalInfo, 1, requestID)

	// STEP 6: Verify final response is ready
	fmt.Printf("📥 Agent Response: %s\n", response2.OutputType)
	if response2.OutputType != OutputReady {
		test.TestPassed = false
		test.Score = 60
		test.FailureReason = fmt.Sprintf("Expected OutputReady after additional info, got %s", response2.OutputType)
		fmt.Printf("❌ FAILED: %s\n", test.FailureReason)
		return test
	}
	fmt.Printf("✅ Correct: Agent provided final output\n")
	fmt.Printf("   Output: \"%s\"\n", truncateString(response2.Output, 80))

	// STEP 7: Verify VLC incremented again
	test.AfterSecondStep = response2.VLCClock.Copy()
	fmt.Printf("📊 VLC After Step 2: %v\n", test.AfterSecondStep)

	if !v.verifyVLCIncrement(test.AfterFirstStep, test.AfterSecondStep, 1, 2) {
		test.TestPassed = false
		test.Score = 70
		test.FailureReason = "VLC did not increment correctly on second response"
		fmt.Printf("❌ FAILED: %s\n", test.FailureReason)
		fmt.Printf("   Expected: increment by 2 on node 1 (message enter + message leave)\n")
		fmt.Printf("   Got: Step1[1]=%d, Step2[1]=%d\n",
			test.AfterFirstStep.Values[1], test.AfterSecondStep.Values[1])
		return test
	}
	fmt.Printf("✅ Correct: VLC incremented properly [node 1: %d → %d] (message enter + message leave)\n",
		test.AfterFirstStep.Values[1], test.AfterSecondStep.Values[1])
	fmt.Println()

	// STEP 8: Verify overall causality
	fmt.Printf("🔐 Verifying causal consistency...\n")
	if !v.verifyCausalConsistency(test.InitialClock, test.AfterFirstStep, test.AfterSecondStep) {
		test.TestPassed = false
		test.Score = 85
		test.FailureReason = "Causal consistency violated"
		fmt.Printf("❌ FAILED: %s\n", test.FailureReason)
		return test
	}
	fmt.Printf("✅ Causal consistency maintained throughout test\n")
	fmt.Println()

	// ALL TESTS PASSED
	test.TestPassed = true
	test.Score = 100

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ VLC VALIDATION PASSED\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Agent: %s\n", miner.ID)
	fmt.Printf("Score: %d/100\n", test.Score)
	fmt.Printf("Status: AUTHORIZED for subnet operations\n")
	fmt.Printf("VLC Implementation: CORRECT\n")
	fmt.Println()

	return test
}

// verifyVLCIncrement checks that a specific node's VLC incremented by expected amount
func (v *CoreValidator) verifyVLCIncrement(before, after *vlc.Clock, nodeID int, expectedIncrement int) bool {
	beforeValue := before.Values[uint64(nodeID)]
	afterValue := after.Values[uint64(nodeID)]
	actualIncrement := int(afterValue - beforeValue)

	return actualIncrement == expectedIncrement
}

// verifyCausalConsistency ensures the VLC sequence maintains causal ordering
func (v *CoreValidator) verifyCausalConsistency(initial, step1, step2 *vlc.Clock) bool {
	// Each step should be causally after the previous
	// step1 should happen-after initial
	// step2 should happen-after step1

	// Simple check: ensure monotonic increase for miner node
	if step1.Values[1] <= initial.Values[1] {
		return false
	}
	if step2.Values[1] <= step1.Values[1] {
		return false
	}

	return true
}

// truncateString truncates a string to maxLen with ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// CreateVLCValidationResult creates a validation result for on-chain submission
func (v *CoreValidator) CreateVLCValidationResult(test *VLCValidationTest) *VLCValidationResult {
	details := "VLC protocol test completed"
	if !test.TestPassed {
		details = fmt.Sprintf("VLC test failed: %s", test.FailureReason)
	}

	return &VLCValidationResult{
		AgentID:     test.AgentID,
		ValidatorID: v.ID,
		Score:       test.Score,
		Passed:      test.TestPassed,
		Details:     details,
		Timestamp:   test.Timestamp,
	}
}

// GetVLCValidationSummary aggregates results from multiple validators
func GetVLCValidationSummary(results []*VLCValidationResult) (avgScore uint8, passed bool) {
	if len(results) == 0 {
		return 0, false
	}

	totalScore := 0
	for _, result := range results {
		totalScore += int(result.Score)
	}

	avgScore = uint8(totalScore / len(results))
	passed = avgScore >= 70 // Pass threshold

	return avgScore, passed
}
