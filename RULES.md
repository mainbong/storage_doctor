# Storage Doctor 개발 규칙

이 문서는 Storage Doctor 프로젝트의 개발 규칙과 패턴을 정의합니다.

## 목차

1. [아키텍처 원칙](#아키텍처-원칙)
2. [의존성 관리](#의존성-관리)
3. [테스트 작성 규칙](#테스트-작성-규칙)
4. [코드 구조](#코드-구조)
5. [네이밍 규칙](#네이밍-규칙)
6. [에러 처리](#에러-처리)

## 아키텍처 원칙

### 1. Dependency Inversion Principle (의존성 역전 원칙)

**모든 외부 의존성은 인터페이스를 통해 추상화해야 합니다.**

- 파일 시스템 작업 → `filesystem.FileSystem` 인터페이스
- HTTP 요청 → `httpclient.HTTPClient` 인터페이스
- 명령어 실행 → `shell.CommandExecutor` 인터페이스
- LLM Provider → `llm.Provider` 인터페이스

**이유**: 테스트 가능성과 모듈 간 결합도를 낮추기 위함

### 2. Dependency Injection (의존성 주입)

**구조체는 생성자를 통해 의존성을 주입받아야 합니다.**

```go
// ✅ 좋은 예
type Manager struct {
    fs filesystem.FileSystem
}

func NewManager(fs filesystem.FileSystem) *Manager {
    return &Manager{fs: fs}
}

// ❌ 나쁜 예
type Manager struct {
    fs *filesystem.OSFileSystem  // 구체 타입에 의존
}
```

## 의존성 관리

### 인터페이스 정의 위치

1. **공통 인터페이스**: `internal/{domain}/` 패키지에 정의
   - `internal/filesystem/filesystem.go`
   - `internal/httpclient/httpclient.go`
   - `internal/shell/executor_interface.go`

2. **도메인별 인터페이스**: 해당 도메인 패키지 내부에 정의
   - `internal/llm/provider.go`
   - `internal/search/provider.go`

### Mock 구현체

**모든 인터페이스는 테스트를 위한 Mock 구현체를 제공해야 합니다.**

- Mock 구현체는 `internal/{domain}/mock.go` 또는 `internal/{domain}/mock_{interface}.go`에 위치
- Mock은 테스트에서만 사용되며, 실제 구현과 동일한 동작을 시뮬레이션해야 함

```go
// internal/filesystem/mock.go
type MockFileSystem struct {
    files map[string][]byte
    // ...
}

func NewMockFileSystem() *MockFileSystem {
    return &MockFileSystem{
        files: make(map[string][]byte),
    }
}
```

## 테스트 작성 규칙

### 1. 테스트 파일 위치

- 모든 테스트 파일은 `{package}_test.go` 형식으로 같은 패키지에 위치
- 예: `config.go` → `config_test.go`

### 2. 테스트 함수 네이밍

- 테스트 함수는 `Test{FunctionName}` 형식
- 에러 케이스는 `Test{FunctionName}_{ErrorCase}` 형식
- 예: `TestLoad`, `TestLoad_ReadError`

### 3. 테스트 구조

```go
func TestFunctionName(t *testing.T) {
    // 1. Setup
    mockFS := filesystem.NewMockFileSystem()
    
    // 2. Execute
    result, err := Function(mockFS)
    
    // 3. Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("expected %v, got %v", expected, result)
    }
}
```

### 4. 테스트 커버리지

- 모든 공개 함수는 테스트되어야 함
- 에러 케이스도 가능한 한 테스트해야 함
- 목표 커버리지: 70% 이상

## 코드 구조

### 패키지 구조

```
internal/
├── {domain}/          # 도메인별 패키지
│   ├── {domain}.go    # 메인 로직
│   ├── {domain}_test.go  # 테스트
│   └── mock.go         # Mock 구현체 (필요시)
├── filesystem/         # 공통 인터페이스
├── httpclient/         # 공통 인터페이스
└── shell/              # 공통 인터페이스
```

### 파일 분리 원칙

- 한 파일에 너무 많은 기능이 있으면 도메인별로 분리
- 테스트 파일은 기능별로 분리 가능 (예: `manager_test.go`, `parser_test.go`)

## 네이밍 규칙

### 생성자 함수

1. **기본 생성자**: `New{Type}()`
   - 실제 구현체를 사용하는 경우
   - 예: `NewManager()`, `NewExecutor()`

2. **테스트용 생성자**: `New{Type}With{Interface}()`
   - 인터페이스를 주입받는 경우
   - 예: `NewManagerWithFS(fs filesystem.FileSystem)`
   - 예: `NewExecutorWithExecutor(exec CommandExecutor)`

```go
// 기본 생성자 (실제 사용)
func NewManager() (*Manager, error) {
    return NewManagerWithFS(filesystem.NewOSFileSystem())
}

// 테스트용 생성자
func NewManagerWithFS(fs filesystem.FileSystem) (*Manager, error) {
    // ...
}
```

### 인터페이스 네이밍

- 인터페이스 이름은 명사형 (예: `FileSystem`, `HTTPClient`)
- Mock 구현체는 `Mock{InterfaceName}` 형식 (예: `MockFileSystem`)

## 에러 처리

### 에러 래핑

**에러는 컨텍스트를 포함하여 래핑해야 합니다.**

```go
// ✅ 좋은 예
if err != nil {
    return nil, fmt.Errorf("failed to read config file: %w", err)
}

// ❌ 나쁜 예
if err != nil {
    return nil, err
}
```

### 에러 검사

**모든 에러는 반드시 검사해야 합니다.**

```go
// ✅ 좋은 예
data, err := fs.ReadFile(path)
if err != nil {
    return err
}

// ❌ 나쁜 예
data, _ := fs.ReadFile(path)  // 에러 무시 금지
```

## 추가 규칙

### 1. Go 표준 라이브러리 우선

- 가능한 한 Go 표준 라이브러리를 사용
- 외부 의존성은 최소화

### 2. 문서화

- 모든 공개 함수는 주석을 작성해야 함
- 복잡한 로직은 인라인 주석으로 설명

### 3. 동시성 안전성

- Mock 구현체는 동시성 안전해야 함 (mutex 사용)
- 실제 구현체도 필요시 동시성 안전 고려

---

**마지막 업데이트**: 2025-01-27

