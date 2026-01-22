//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// ================== JavaScript Tests ==================

func TestExtractJSFunction(t *testing.T) {
	content := `function fetchUser(userId) {
	return api.get('/users/' + userId);
}

function syncFunc(a, b) {
	return a + b;
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.js", content)
	testFile := filepath.Join(tmpDir, "test.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数シンボルを探す
	var functions []Symbol
	for _, s := range fileSymbols.Symbols {
		if s.Kind == "function" {
			functions = append(functions, s)
		}
	}

	if len(functions) < 2 {
		t.Fatalf("Expected at least 2 function symbols, got %d", len(functions))
	}

	// fetchUser関数
	fetchUser := functions[0]
	if fetchUser.Name != "fetchUser" {
		t.Errorf("Expected 'fetchUser', got '%s'", fetchUser.Name)
	}
	// 引数が含まれているか確認
	if !strings.Contains(fetchUser.Signature, "userId") {
		t.Errorf("Expected signature to contain 'userId', got '%s'", fetchUser.Signature)
	}
}

func TestExtractJSClass(t *testing.T) {
	content := `class User {
	constructor(name) {
		this.name = name;
	}

	greet() {
		return "Hello, " + this.name;
	}
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "user.js", content)
	testFile := filepath.Join(tmpDir, "user.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// class + constructor + greet = 3 symbols
	if len(fileSymbols.Symbols) < 1 {
		t.Fatalf("Expected at least 1 symbol, got %d", len(fileSymbols.Symbols))
	}

	// User class
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "class" {
		t.Errorf("Expected 'class', got '%s'", user.Kind)
	}
}

// ================== TypeScript Tests ==================

// Issue #58: TypeScript型注釈抽出テスト
func TestExtractTypeScriptFunction(t *testing.T) {
	content := `function fetchUser(userId: string, options?: RequestOptions): Promise<User> {
	return api.get('/users/' + userId);
}

async function createUser(name: string, age: number): Promise<User> {
	return api.post('/users', { name, age });
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.ts", content)
	testFile := filepath.Join(tmpDir, "test.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 関数シンボルを探す
	var functions []Symbol
	for _, s := range fileSymbols.Symbols {
		if s.Kind == "function" {
			functions = append(functions, s)
		}
	}

	if len(functions) < 2 {
		t.Fatalf("Expected at least 2 function symbols, got %d", len(functions))
	}

	// fetchUser関数: 型注釈が含まれているか
	fetchUser := functions[0]
	if fetchUser.Name != "fetchUser" {
		t.Errorf("Expected 'fetchUser', got '%s'", fetchUser.Name)
	}
	// パラメータ型注釈
	if !strings.Contains(fetchUser.Signature, "string") {
		t.Errorf("Expected signature to contain type 'string', got '%s'", fetchUser.Signature)
	}
	// 戻り値型注釈
	if !strings.Contains(fetchUser.Signature, "Promise<User>") {
		t.Errorf("Expected signature to contain return type 'Promise<User>', got '%s'", fetchUser.Signature)
	}

	// createUser関数: async + 型注釈
	createUser := functions[1]
	if createUser.Name != "createUser" {
		t.Errorf("Expected 'createUser', got '%s'", createUser.Name)
	}
	// async キーワード
	if !strings.Contains(createUser.Signature, "async") {
		t.Errorf("Expected signature to contain 'async', got '%s'", createUser.Signature)
	}
}

// Issue #58: TypeScriptメソッド型注釈テスト
func TestExtractTypeScriptMethod(t *testing.T) {
	content := `class UserService {
	async getUser(id: string): Promise<User> {
		return this.api.get(id);
	}

	updateUser(id: string, data: UserData): User {
		return this.api.put(id, data);
	}
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "service.ts", content)
	testFile := filepath.Join(tmpDir, "service.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// メソッドシンボルを探す
	var methods []Symbol
	for _, s := range fileSymbols.Symbols {
		if s.Kind == "method" {
			methods = append(methods, s)
		}
	}

	if len(methods) < 2 {
		t.Fatalf("Expected at least 2 method symbols, got %d", len(methods))
	}

	// getUser: async + 型注釈
	getUser := methods[0]
	if getUser.Name != "getUser" {
		t.Errorf("Expected 'getUser', got '%s'", getUser.Name)
	}
	if !strings.Contains(getUser.Signature, "async") {
		t.Errorf("Expected signature to contain 'async', got '%s'", getUser.Signature)
	}
	if !strings.Contains(getUser.Signature, "string") {
		t.Errorf("Expected signature to contain 'string', got '%s'", getUser.Signature)
	}
}

// ================== React Arrow Function / Hooks Tests ==================

func TestExtractArrowFunctionComponent(t *testing.T) {
	// Note: JSX構文はTypeScriptパーサーで正しく解析されないため、
	// JSXなしの形式でテスト
	content := `const Header = () => {
    return null;
};

const Button: React.FC<Props> = ({ onClick }) => {
    return null;
};

const App = () => {
    return "Hello";
};
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "components.ts", content)
	testFile := filepath.Join(tmpDir, "components.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// コンポーネントを探す
	components := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "component" {
			components[sym.Name] = sym
		}
	}

	// Header コンポーネント
	if _, ok := components["Header"]; !ok {
		t.Error("Expected to find 'Header' component")
	}

	// Button コンポーネント（型注釈付き）
	button, ok := components["Button"]
	if !ok {
		t.Error("Expected to find 'Button' component")
	} else {
		if !strings.Contains(button.Signature, "React.FC<Props>") {
			t.Errorf("Expected Button signature to contain 'React.FC<Props>', got '%s'", button.Signature)
		}
	}

	// App コンポーネント
	if _, ok := components["App"]; !ok {
		t.Error("Expected to find 'App' component")
	}
}

func TestExtractArrowFunctionHook(t *testing.T) {
	content := `const useAuth = () => {
    const [user, setUser] = useState(null);
    return { user, setUser };
};

const useCounter = (initial: number) => {
    const [count, setCount] = useState(initial);
    return { count, increment: () => setCount(c => c + 1) };
};
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "hooks.tsx", content)
	testFile := filepath.Join(tmpDir, "hooks.tsx")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// Hookを探す
	hooks := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		if sym.Kind == "hook" {
			hooks[sym.Name] = sym
		}
	}

	// useAuth Hook
	if _, ok := hooks["useAuth"]; !ok {
		t.Error("Expected to find 'useAuth' hook")
	}

	// useCounter Hook
	useCounter, ok := hooks["useCounter"]
	if !ok {
		t.Error("Expected to find 'useCounter' hook")
	} else {
		if !strings.Contains(useCounter.Signature, "(initial: number)") {
			t.Errorf("Expected useCounter signature to contain '(initial: number)', got '%s'", useCounter.Signature)
		}
	}
}

func TestExtractFunctionDeclarationHook(t *testing.T) {
	content := `function useAuth() {
    const [user, setUser] = useState(null);
    return { user, setUser };
}

function useCounter(initial) {
    const [count, setCount] = useState(initial);
    return { count };
}

function App() {
    return <div>Hello</div>;
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "hooks.js", content)
	testFile := filepath.Join(tmpDir, "hooks.js")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// シンボルのKindを確認
	kindMap := make(map[string]string)
	for _, sym := range fileSymbols.Symbols {
		kindMap[sym.Name] = sym.Kind
	}

	// useAuth は hook
	if kindMap["useAuth"] != "hook" {
		t.Errorf("Expected useAuth to be 'hook', got '%s'", kindMap["useAuth"])
	}

	// useCounter は hook
	if kindMap["useCounter"] != "hook" {
		t.Errorf("Expected useCounter to be 'hook', got '%s'", kindMap["useCounter"])
	}

	// App は function（PascalCaseだがfunction宣言なのでfunctionのまま）
	if kindMap["App"] != "function" {
		t.Errorf("Expected App to be 'function', got '%s'", kindMap["App"])
	}
}

func TestArrowFunctionSkipsRegularVariables(t *testing.T) {
	content := `const handler = () => {
    console.log("clicked");
};

const callback = (x) => x * 2;

const Header = () => {
    return null;
};
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "mixed.ts", content)
	testFile := filepath.Join(tmpDir, "mixed.ts")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	// 名前でシンボルをマップ
	symbolMap := make(map[string]Symbol)
	for _, sym := range fileSymbols.Symbols {
		symbolMap[sym.Name] = sym
	}

	// handler と callback はスキップされるべき（小文字始まり、hookでもない）
	if _, ok := symbolMap["handler"]; ok {
		t.Error("Expected 'handler' to be skipped (not a component or hook)")
	}
	if _, ok := symbolMap["callback"]; ok {
		t.Error("Expected 'callback' to be skipped (not a component or hook)")
	}

	// Header はコンポーネントとして抽出される
	if sym, ok := symbolMap["Header"]; !ok {
		t.Error("Expected to find 'Header' component")
	} else if sym.Kind != "component" {
		t.Errorf("Expected Header kind to be 'component', got '%s'", sym.Kind)
	}
}
