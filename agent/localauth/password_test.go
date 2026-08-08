package localauth

import "testing"

var testParams = Argon2Params{
	Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16,
}

func TestHashAndVerifyPassword(t *testing.T) {
	verifier, err := HashPassword("correct horse", testParams)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := VerifyPassword("correct horse", verifier)
	if err != nil || !valid {
		t.Fatalf("correct password valid=%t err=%v", valid, err)
	}
	valid, err = VerifyPassword("wrong horse", verifier)
	if err != nil || valid {
		t.Fatalf("wrong password valid=%t err=%v", valid, err)
	}
}

func TestVerifyPasswordRejectsUnsafeParameters(t *testing.T) {
	verifier := "$argon2id$v=19$m=999999999,t=3,p=2$c2FsdHNhbHQ$aGFzaGhhc2hoYXNoaGFzaA"
	if _, err := VerifyPassword("password", verifier); err == nil {
		t.Fatal("unsafe verifier should fail before allocating memory")
	}
}
